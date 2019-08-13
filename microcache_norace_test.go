// +build !race

package microcache

import (
	"net/http"
	"testing"
	"time"
)

// StaleWhilRevalidate
func TestStaleWhilRevalidate(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:                  30 * time.Second,
		StaleWhileRevalidate: 30 * time.Second,
		Monitor:              testMonitor,
		Driver:               NewDriverLRU(10),
	})
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))

	// prime cache
	batchGet(handler, []string{
		"/",
		"/",
	})
	if testMonitor.misses != 1 || testMonitor.hits != 1 {
		t.Log("StaleWhilRevalidate not respected - got", testMonitor.misses, "misses")
		t.Fail()
	}

	// stale and hit after 30s
	cache.offsetIncr(30 * time.Second)
	batchGet(handler, []string{
		"/",
	})
	time.Sleep(10 * time.Millisecond)
	batchGet(handler, []string{
		"/",
	})
	if testMonitor.stales != 1 || testMonitor.hits != 2 {
		t.Log("StaleWhilRevalidate not respected - got", testMonitor.stales, "stales")
		t.Fail()
	}
	cache.Stop()
}

// CollapsedFowarding and StaleWhileRevalidate
func TestCollapsedFowardingStaleWhileRevalidate(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:                  30 * time.Second,
		CollapsedForwarding:  true,
		StaleWhileRevalidate: 30 * time.Second,
		Monitor:              testMonitor,
		Driver:               NewDriverLRU(10),
	})
	handler := cache.Middleware(http.HandlerFunc(timelySuccessHandler))
	batchGet(handler, []string{
		"/",
	})
	cache.offsetIncr(31 * time.Second)
	start := time.Now()
	parallelGet(handler, []string{
		"/",
		"/",
		"/",
		"/",
		"/",
		"/",
	})
	end := time.Since(start)
	// Sleep for a little bit to give the StaleWhileRevalidate goroutines some time to start.
	time.Sleep(time.Millisecond * 10)
	if testMonitor.misses != 1 || testMonitor.stales != 6 || testMonitor.backend != 2 || end > 20*time.Millisecond {
		t.Logf("%#v", testMonitor)
		t.Log("CollapsedFowarding and StaleWhileRevalidate not respected - got", testMonitor.backend, "backend")
		t.Fail()
	}
	cache.Stop()
}
