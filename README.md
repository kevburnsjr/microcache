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
microcache deployments make use of existing HTTP caching middleware like NGINX. This
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
	"bytes"
	"log"
	"net/http"
	"time"

	"github.com/kevburnsjr/microcache"
)

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

type handler struct {
}

var body = bytes.Repeat([]byte("1234567890"), 1e3)

// This example fills up to 1.2GB of memory, so at least 2.0GB of RAM is recommended
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Enable cache
	w.Header().Set("microcache-cache", "1")

	// Return a 10 kilobyte response body
	w.Write(body)
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
```

## Features

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
```

## Benchmarks

All benchmarks are lies. Running example code above on 5820k i7 @ 3.9Ghz DDR4.
GOMAXPROCS=2, 10KB response.

```
> gobench -u http://localhost/ -c 10 -t 10
Dispatching 10 clients
Waiting for results...

Requests:                           404120 hits
Successful requests:                404120 hits
Network failed:                          0 hits
Bad requests failed (!2xx):              0 hits
Successful requests rate:            40412 hits/sec
Read throughput:                 410714568 bytes/sec
Write throughput:                  3273453 bytes/sec
Test time:                              10 sec
```

The intent of this middleware is to serve cached content with minimal overhead. We could
probably do some more gymnastics to reduce allocs but overall performance is quite good.
It achieves the goal of reducing hit response times to the order of microseconds.

```
$ go test -bench=. -benchmem
goos: linux
goarch: amd64
pkg: github.com/kevburnsjr/microcache
BenchmarkHits-12                            827476   1449 ns/op    678 B/op   14 allocs/op
BenchmarkNocache-12                        1968576    613 ns/op    312 B/op    6 allocs/op
BenchmarkMisses-12                          307652   3890 ns/op   1765 B/op   37 allocs/op
BenchmarkCompression1kHits-12               648564   1928 ns/op   1384 B/op   15 allocs/op
BenchmarkCompression1kNocache-12           1965351    611 ns/op    312 B/op    6 allocs/op
BenchmarkCompression1kMisses-12             232980   5119 ns/op   3365 B/op   39 allocs/op
BenchmarkParallelCompression1kHits-12      2525994    511 ns/op   1383 B/op   15 allocs/op
BenchmarkParallelCompression1kNocache-12   3876728    335 ns/op    312 B/op    6 allocs/op
BenchmarkParallelCompression1kMisses-12     666580   2226 ns/op   3326 B/op   38 allocs/op
PASS
ok      github.com/kevburnsjr/microcache        7.188s
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
