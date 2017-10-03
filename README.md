microcache is a non-standard HTTP microcache implemented as Go middleware.

HTTP [Microcaching](https://www.nginx.com/blog/benefits-of-microcaching-nginx/)
is a proven strategy for improving the efficiency, availability, consistency and
response time variability of HTTP web services. These benefits are especially relevant
in microservice architectures where a service's synchronous dependencies sometimes
become unavailable and it is not always feasible or economical to add a caching layer
between all services.

To date, very few software packages exist to solve this specific problem. Most
microcache deployments make use of existing HTTP caching middleware. This presents
a challenge. When an HTTP cache exists for the purpose of microcaching between an
origin server and a CDN, the server must choose whether to use standard HTTP caching
headers with aggressive short TTLs for the microcache or less aggressive longer TTL
headers more suitable to CDNs. The overlap in HTTP header key space prevents these two
cache layers from coexisting without some additional customization.

All request specific custom headers supported by this cache are prefixed with
```microcache-``` and scrubbed from the response. Most of the common HTTP caching
headers one would expect to see in an http cache are ignored (except Vary). This
was intentional and support may change depending on developer feedback. The purpose of
this cache is not to act as a substitute for a robust HTTP caching layer but rather
to serve as an additional caching layer with separate controls for shorter lived,
more aggressive caching measures.

The manner in which this cache operates (writing responses to byte buffers) may not be
suitable for all applications. Caching should certainly be disabled for any resources
serving very large and/or streaming responses. For instance, caching is automatically
disabled for all websocket requests.

More info in the docs: https://godoc.org/github.com/httpimp/microcache

## Example

```go
package main

import (
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/justinas/alice"

	"github.com/httpimp/microcache"
)

type handler struct {
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Enable cache
	w.Header().Set("microcache-cache", "1")

	// Return a response body of random size between 10 and 100 kilobytes
	// Requests per sec for cache hits is mostly dependent on response size
	// This cache can saturate a gigabit network connection with cache hits
	// containing response bodies as small as 10kb on a dual core 3.3 Ghz i7 VM
	n := rand.Intn(10)*1e4 + 1e4
	msg := strings.Repeat("1234567890", n)
	http.Error(w, msg, 200)

}

func logStats(stats microcache.Stats) {
	total := stats.Hits + stats.Misses + stats.Stales
	log.Printf("Size: %d, Total: %d, Hits: %d, Misses: %d, Stales: %d, Errors: %d\n",
		stats.Size,
		total,
		stats.Hits,
		stats.Misses,
		stats.Stales,
		stats.Errors,
	)
}

func main() {
	// - Nocache: true
	// Cache is disabled for all requests by default
	// Cache can be enabled per endpoint with response header
	//
	//     microcache-cache: 1
	//
	// - Timeout: 5 * time.Second
	// Requests will be timed out and treated as 503 if they do not return within 5s
	//
	// - TTL: 10 * time.Second
	// Responses which enable cache explicitly will be cached for 10s by default
	// Response cache time can be configured per endpoint with response header
	//
	//     microcache-ttl: 30
	//
	// - StaleIfError: 600 * time.Second
	// If the request encounters an error (or times out), a stale response will be returned
	// provided that the stale cached response expired less than 10 minutes ago.
	// Can be altered per endpoint with response header
	// More Info: https://tools.ietf.org/html/rfc5861
	//
	//     microcache-stale-if-error: 86400
	//
	// - StaleRecache: true
	// Upon serving a stale response following an error, that stale response will be
	// re-cached for the default ttl (10s)
	// Can be disabled per endpoint with response header
	//
	//     microcache-no-stale-recache: 1
	//
	// - StaleWhileRevalidate: 20 * time.Second
	// If the cache encounters a request for a cached object that has expired in the
	// last 20s, the cache will reply immediately with a stale response and fetch
	// the resource in a background process.
	// More Info: https://tools.ietf.org/html/rfc5861
	//
	//     microcache-stale-while-revalidate: 20
	//
	// - TTLSync: true
	// Cache expiration times will be synced to the system clock to avoid inconsistency
	// between caches. In effect, all expiration times will fall on a multiple of 10s
	// Can be disabled per endpoint with response header
	//
	//     microcache-ttl-nosync: 1
	//
	// - HashQuery: false
	// Query parameters are not hashed by default
	// Responses can be splintered by query parameter with response header
	//
	//     microcache-vary-query: page, limit, etc...
	//
	// - Exposed: true
	// Header will be appended to response indicating HIT / MISS / STALE
	//
	//     microcache: ( HIT | MISS | STALE )
	//
	// - Monitor: microcache.MonitorFunc(5 * time.Second, logStats)
	// LogStats will be called every 5s to log stats about the cache
	//
	cache := microcache.New(microcache.Config{
		Nocache:              true,
		Timeout:              5 * time.Second,
		TTL:                  10 * time.Second,
		StaleIfError:         600 * time.Second,
		StaleRecache:         true,
		StaleWhileRevalidate: 20 * time.Second,
		TTLSync:              true,
		HashQuery:            false,
		Exposed:              true,
		Monitor:              microcache.MonitorFunc(5*time.Second, logStats),
	})

	chain := alice.New(cache.Middleware)
	h := chain.Then(handler{})

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

May improve cache consistency

* **ttl-sync** - synchronize response expiration times

Supports content negotiation with global and request specific cache splintering

* **vary** - splinter responses by request header value
* **vary-query** - splinter responses by URL query parameter value

## Release

Tests have not yet been written to confirm the correct behavior of this cache.

While it is fairly certain that all logic pertaining to the various caching mechanisms
that this cache supports is operating correctly, 100% test coverage should be completed
before this library can be recommended for use in production.

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

## Notes

```
Move DriverGcache to microcache/driver package

Vary query by parameter presence as well as value

Modify Monitor.Error to accept request, response and error
Add Monitor.Timeout accepting request, response and error

Separate middleware:
  Sanitize lang header? (first language)
  Sanitize region? (country code)

gzip cache entries?
etag support?
if-modified-since support?
HTCP?
TCI?
Custom rule handling?
  Passthrough: func(r) bool
```
