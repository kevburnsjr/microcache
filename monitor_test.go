package microcache

import (
	"testing"
	"time"
)

// Remove should work as expected
func TestMonitor(t *testing.T) {
	var hits int
	var expected = 4
	testMonitor := MonitorFunc(100 * time.Second, func(s Stats) {
		hits = s.Hits
	})
	testMonitor.hits = int64(expected)
	testMonitor.Log(Stats{})
	if hits != expected {
		t.Fatalf("Monitor not logging correctly (%d != %d)", hits, expected)
	}
}
