# Microcache

A non-standard HTTP cache implemented as Go middleware.

[![GoDoc](https://godoc.org/github.com/kevburnsjr/microcache?status.svg)](https://godoc.org/github.com/kevburnsjr/microcache)
[![Go Report Card](https://goreportcard.com/badge/github.com/kevburnsjr/microcache?1)](https://goreportcard.com/report/github.com/kevburnsjr/microcache)
[![Code Coverage](http://gocover.io/_badge/github.com/kevburnsjr/microcache?1)](http://gocover.io/github.com/kevburnsjr/microcache)

HTTP [Microcaching](https://www.nginx.com/blog/benefits-of-microcaching-nginx/)
is a common strategy for improving the efficiency, availability and
response time variability of HTTP web services. These benefits are especially relevant
in microservice architectures where a service's synchronous dependencies sometimes
become unavailable and it is not always feasible or economical to add a separate
caching layer between all services.

To date, very few software packages exist to solve this specific problem. Most
microcache deployments make use of existing HTTP caching middleware like NIGNX. This
presents a challenge. When an HTTP cache exists for the purpose of microcaching between
an origin server and a CDN, the origin must choose whether to use standard HTTP caching
headers with aggressive short TTLs for the microcache or less aggressive longer TTL
headers more suitable to CDNs. The overlap in HTTP header key space prevents these two
cache layers from coexisting without some additional customization.

All request specific custom response headers supported by this cache are prefixed with
```microcache-``` and scrubbed from the response. Most of the common HTTP caching
headers one would expect to see in an http cache are ignored (except Vary). This
was intentional and support may change depending on developer feedback. The purpose of
this cache is not to act as a substitute for a robust HTTP caching layer but rather
to serve as an additional caching layer with separate controls for shorter lived,
more aggressive caching measures.

The manner in which this cache operates (writing response bodies to byte buffers) may
not be suitable for all applications. Caching should certainly be disabled for any
resources serving very large and/or streaming responses. For instance, caching is
automatically disabled for all websocket requests.

More info in the docs: https://godoc.org/github.com/kevburnsjr/microcache

## Example

```go
package main

import (
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/kevburnsjr/microcache"
)

type handler struct {
}

// This example fills up to 1.2GB of memory, so at least 2.0GB of RAM is recommended
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Enable cache
	w.Header().Set("microcache-cache", "1")

	randn := rand.Intn(10) + 1

	// Sleep between 10 and 100 ms
	time.Sleep(time.Duration(randn*10) * time.Millisecond)

	// Return a response body of random size between 10 and 100 kilobytes
	// Requests per sec for cache hits is mostly dependent on response size
	// This cache can saturate a gigabit network connection with cache hits
	// containing response bodies as small as 10kb on a dual core 3.3 Ghz i7 VM
	http.Error(w, strings.Repeat("1234567890", randn*1e3), 200)
}

func logStats(stats microcache.Stats) {
	total := stats.Hits + stats.Misses + stats.Stales
	log.Printf("Size: %d, Total: %d, Hits: %d, Misses: %d, Stales: %d, Backend: %d, Errors: %d\n",
		stats.Size,
		total,
		stats.Hits,
		stats.Misses,
		stats.Stales,
		stats.Backend,
		stats.Errors,
	)
}

func main() {
	cache := microcache.New(microcache.Config{
		Nocache:              true,
		Timeout:              3 * time.Second,
		TTL:                  30 * time.Second,
		StaleIfError:         3600 * time.Second,
		StaleRecache:         true,
		StaleWhileRevalidate: 30 * time.Second,
		CollapsedForwarding:  true,
		HashQuery:            true,
		QueryIgnore:          []string{},
		Exposed:              true,
		SuppressAgeHeader:    false,
		Monitor:              microcache.MonitorFunc(5*time.Second, logStats),
		Driver:               microcache.NewDriverLRU(1e4),
		Compressor:           microcache.CompressorSnappy{},
	})
	defer cache.Stop()

	h := cache.Middleware(handler{})

	http.ListenAndServe(":80", h)
}
```

## Benefits

May improve service efficiency by reducing origin read traffic

* **ttl** - response caching with global or request specific ttl
* **collapsed-forwarding** - deduplicate requests for cacheable resources

May improve client facing response time variability

* **stale-while-revalidate** - serve stale content while fetching cacheable resources in the background

May improve service availability

* **request-timeout** - kill long running requests
* **stale-if-error** - serve stale responses on error (or request timeout)
* **stale-recache** - recache stale responses following stale-if-error

Supports content negotiation with global and request specific cache splintering

* **vary** - splinter requests by request header value
* **vary-query** - splinter requests by URL query parameter value

## Control Flow Diagram

This diagram illustrates the basic internal operation of the middleware.

![microcache-architecture.svg](docs/microcache-architecture.svg)

## Compression

The Snappy compressor is recommended to optimize for CPU over memory efficiency compared with gzip

[Snappy](https://github.com/golang/snappy) provides:

- 14x faster compression over gzip
- 8x faster expansion over gzip
- but the result is 1.5 - 2x the size compared to gzip (for specific json examples)

Your mileage may vary. See [compare_compression.go](tools/compare_compression/compare_compression.go) to test your specific workloads

```
> go run tools/compare_compression.go -f large.json
Original: 616,611 bytes of json
zlib   compress 719.853807ms  61,040 bytes (10.1x)
gzip   compress 720.731066ms  61,052 bytes (10.1x)
snappy compress 48.836002ms  106,613 bytes (5.8x)
zlib   expand 211.538416ms
gzip   expand 220.011961ms
snappy expand 26.973263ms

> go run tools/compare_compression.go -f medium.json
Original: 279,368 bytes of json
zlib   compress 282.549098ms 19,825 bytes (14.1x)
gzip   compress 275.961026ms 19,837 bytes (14.1x)
snappy compress 16.452706ms  37,096 bytes (7.5x)
zlib   expand 86.704103ms
gzip   expand 81.188856ms
snappy expand 10.557594ms

> go run tools/compare_compression.go -f small.json
Original: 53,129 bytes of json
zlib   compress 73.204418ms 5,084 bytes (10.5x)
gzip   compress 74.150401ms 5,096 bytes (10.4x)
snappy compress 5.225558ms  8,412 bytes (6.3x)
zlib   expand 18.764693ms
gzip   expand 18.797717ms
snappy expand 2.354814ms
```

## Benchmarks

All benchmarks are lies. Dual core 3.3Ghz i7 DDR4 Centos 7 VM w/ 10KB response (see example above)

```
> gobench -u http://localhost/ -c 10 -t 10
Dispatching 10 clients
Waiting for results...

Requests:                           110705 hits
Successful requests:                110705 hits
Network failed:                          0 hits
Bad requests failed (!2xx):              0 hits
Successful requests rate:            11070 hits/sec
Read throughput:                1109430818 bytes/sec
Write throughput:                   896791 bytes/sec
Test time:                              10 sec
```

## Release Status

API is stable. 100% test coverage.

At least one large scale deployment of this library has been running in production
on a high volume internet facing API at an Alexa Top 500 global website for over a year.

## Notes

```
Vary query by parameter presence as well as value

Modify Monitor.Error to accept request, response and error
Add Monitor.Timeout accepting request, response and error

Separate middleware:
  Sanitize lang header? (first language)
  Sanitize region? (country code)

etag support?
if-modified-since support?
HTCP?
TCI?
Custom rule handling?
  Passthrough: func(r) bool
```
