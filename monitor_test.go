package microcache

import (
	"net/http"
	"testing"
	"time"
)

// Remove should work as expected
func TestMonitor(t *testing.T) {
	var hits int
	var expected = 4
	testMonitor := MonitorFunc(100*time.Second, func(s Stats) {
		hits = s.Hits
	})
	testMonitor.hits = int64(expected)
	testMonitor.Log(Stats{})
	if hits != expected {
		t.Fatalf("Monitor not logging correctly (%d != %d)", hits, expected)
	}
}

// Microcache calls monitor
func TestMicrocacheCallsMonitor(t *testing.T) {
	var statChan = make(chan int)
	testMonitor := &monitorFunc{interval: 10 * time.Millisecond, logFunc: func(s Stats) {
		statChan <- s.Size
	}}
	cache := New(Config{
		TTL:     30 * time.Second,
		Monitor: testMonitor,
		Driver:  NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{"/"})
	size := <-statChan
	if size != 1 {
		t.Fatal("Monitor was not called by microcache")
	}
}
