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

// This example fills up to 1.2GB of memory, so at least 2.0GB of RAM is recommended
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Enable cache
	w.Header().Set("microcache-cache", "1")

	// Return a response body of random size between 10 and 100 kilobytes
	// Requests per sec for cache hits is mostly dependent on response size
	// This cache can saturate a gigabit network connection with cache hits
	// containing response bodies as small as 10kb on a dual core 3.3 Ghz i7 VM
	randn := rand.Intn(10) + 1

	// Sleep between 10 and 100 ms
	time.Sleep(time.Duration(randn*10) * time.Millisecond)

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
	// - Nocache: true
	// Cache is disabled for all requests by default
	// Cache can be enabled per request hash with response header
	//
	//     microcache-cache: 1
	//
	// - Timeout: 3 * time.Second
	// Requests will be timed out and treated as 503 if they do not return within 35s
	//
	// - TTL: 30 * time.Second
	// Responses which enable cache explicitly will be cached for 30s by default
	// Response cache time can be configured per endpoint with response header
	//
	//     microcache-ttl: 30
	//
	// - StaleIfError: 3600 * time.Second
	// If the request encounters an error (or times out), a stale response will be returned
	// provided that the stale cached response expired less than an hour ago.
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
	// - StaleWhileRevalidate: 30 * time.Second
	// If the cache encounters a request for a cached object that has expired in the
	// last 30s, the cache will reply immediately with a stale response and fetch
	// the resource in a background process.
	// More Info: https://tools.ietf.org/html/rfc5861
	//
	//     microcache-stale-while-revalidate: 20
	//
	// - HashQuery: true
	// All query parameters are included in the request hash
	//
	// - QueryIgnore: []string{}
	// A list of query parameters to ignore when hashing the request
	// Add oauth parameters or other unwanted cache busters to this list
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
		Timeout:              3 * time.Second,
		TTL:                  30 * time.Second,
		StaleIfError:         3600 * time.Second,
		StaleRecache:         true,
		StaleWhileRevalidate: 30 * time.Second,
		CollapsedForwarding:  true,
		HashQuery:            true,
		QueryIgnore:          []string{},
		Exposed:              true,
		Monitor:              microcache.MonitorFunc(5*time.Second, logStats),
		Driver:               microcache.NewDriverLRU(1e4),
		Compressor:           microcache.CompressorSnappy{},
	})

	chain := alice.New(cache.Middleware)
	h := chain.Then(handler{})

	http.ListenAndServe(":80", h)
}
