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
