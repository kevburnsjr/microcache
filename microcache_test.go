package microcache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// QueryIgnore should be respected when HashQuery is true
func TestQueryIgnore(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:         30 * time.Second,
		HashQuery:   true,
		QueryIgnore: []string{"a"},
		Monitor:     testMonitor,
	})
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
		"/?a=1",
		"/?a=2",
	})
	if testMonitor.misses != 1 || testMonitor.hits != 2 {
		t.Log("Query parameters not ignored - got", testMonitor.misses, "misses")
		t.Fail()
	}
	cache.Stop()
}

// QueryIgnore should be disregarded when HashQuery is false
func TestQueryIgnoreDisabled(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:         30 * time.Second,
		HashQuery:   false,
		QueryIgnore: []string{"a"},
		Monitor:     testMonitor,
	})
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
		"/?a=1",
		"/?b=2",
	})
	if testMonitor.misses != 1 {
		t.Log("Query parameters ignored - got", testMonitor.misses, "misses")
		t.Fail()
	}
	cache.Stop()
}

// --- helper funcs ---

func batchGet(handler http.Handler, urls []string) {
	for _, url := range urls {
		r1, _ := http.NewRequest("GET", url, nil)
		handler.ServeHTTP(httptest.NewRecorder(), r1)
	}
}

func noopSuccessHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "done", 200)
}
