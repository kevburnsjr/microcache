microcache is an HTTP cache implemented as Go middleware.

Useful for APIs which serve large numbers of identical responses.
Especially useful in high traffic read heavy APIs with common responses
and low sensitivity to freshness.

May improve efficiency by reducing read traffic through

* **ttl** - response caching with global or request specific ttl
* **collapsed-forwarding** - deduplicate requests for cacheable resources

May Improve client facing response time variability with

* **stale-while-revalidate** - serve stale content while fetching cacheable resources

May improve service availability with support for

* **request-timeout** - kill long running requests
* **stale-if-error** - serve stale responses on error (or request timeout)

Supports content negotiation with global and request specific cache splintering

* **vary** - splinter responses by request header value
* **vary-query** - splinter responses by URL query parameter value

Helps maintain cache consistency with

* **ttl-sync** - synchronize TTLs across a load balanced cluster

The above mentioned availability gains are especially relevant in microservices where
synchronous dependencies sometimes become unavailable and it isn't always feasible or
economical to add a caching layer between services (hence the name)

All request specific custom headers supported by the cache are prefixed with
```microcache-``` and scrubbed from the response. Most of the common HTTP caching
headers you would expect to see in an http cache are ignored (except Vary). This
was intentional and may change depending on developer feedback. The purpose of this
cache is not to act as a substitute for a robust HTTP caching layer but rather
to serve as an additional caching layer with separate controls.

The manner in which this cache operates (writing responses to byte buffers) may not be
suitable for all applications. Caching should certainly be disabled for any resources
serving very large and/or streaming responses. For instance, the cache is automatically
disabled for all websocket requests.

## Docs

https://godoc.org/github.com/httpimp/microcache

## Example usage

```go
package main

import (
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
	w.Header().Set("microcache-cache", "1")
	// Print 10b - 100kb of data
	n := rand.Intn(10000) + 1
	msg := strings.Repeat("1234567890", n)
	http.Error(w, msg, 200)
}

func main() {
	cache := microcache.New(microcache.Config{
		Timeout:              2 * time.Second,
		TTL:                  10 * time.Second,
		StaleIfError:         20 * time.Second,
		StaleWhileRevalidate: 10 * time.Second,
		TTLSync:              true,
		Nocache:              true,
		Exposed:              true,
	})

	chain := alice.New(cache.Middleware)
	h := chain.Then(handler{})

	http.ListenAndServe(":80", h)
}
```

## Benchmarks

All benchmarks are lies. Quad core i7 Centos 7 VM w/ caching enabled (code above)

```
> gobench -u http://localhost/ -c 10 -t 10
Dispatching 10 clients
Waiting for results...

Requests:                           303459 hits
Successful requests:                303459 hits
Network failed:                          0 hits
Bad requests failed (!2xx):              0 hits
Successful requests rate:            30345 hits/sec
Read throughput:                 573895881 bytes/sec
Write throughput:                  2458115 bytes/sec
Test time:                              10 sec
```

## Notes

```
Separate middleware:
  Sanitize lang header? (first language)
  Sanitize region? (country code)

etag support?
if-modified-since support?
HTCP?
TCI?
Special rule functions?
  Passthrough: func(req) bool
```
