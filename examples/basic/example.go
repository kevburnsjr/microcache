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
	// Cache can be enabled per request with response header
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
	// Can be altered per request with response header
	// More Info: https://tools.ietf.org/html/rfc5861
	//
	//     microcache-stale-if-error: 86400
	//
	// - StaleRecache: true
	// Upon serving a stale response following an error, that stale response will be
	// re-cached for the default ttl (10s)
	// Can be disabled per request with response header
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
	// Can be disabled per request with response header
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
