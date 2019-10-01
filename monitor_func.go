package microcache

import (
	"sync/atomic"
	"time"
)

// MonitorFunc turns a function into a Monitor
func MonitorFunc(interval time.Duration, logFunc func(Stats)) *monitorFunc {
	return &monitorFunc{
		interval: interval,
		logFunc:  logFunc,
	}
}

type monitorFunc struct {
	interval time.Duration
	logFunc  func(Stats)
	hits     int64
	misses   int64
	stales   int64
	backend  int64
	errors   int64
	stop     chan bool
}

func (m *monitorFunc) GetInterval() time.Duration {
	return m.interval
}

func (m *monitorFunc) Log(stats Stats) {
	// hits
	stats.Hits = int(atomic.SwapInt64(&m.hits, 0))

	// misses
	stats.Misses = int(atomic.SwapInt64(&m.misses, 0))

	// stales
	stats.Stales = int(atomic.SwapInt64(&m.stales, 0))

	// backend
	stats.Backend = int(atomic.SwapInt64(&m.backend, 0))

	// errors
	stats.Errors = int(atomic.SwapInt64(&m.errors, 0))

	// log
	m.logFunc(stats)
}

func (m *monitorFunc) Hit() {
	atomic.AddInt64(&m.hits, 1)
}

func (m *monitorFunc) Miss() {
	atomic.AddInt64(&m.misses, 1)
}

func (m *monitorFunc) Stale() {
	atomic.AddInt64(&m.stales, 1)
}

func (m *monitorFunc) Backend() {
	atomic.AddInt64(&m.backend, 1)
}

func (m *monitorFunc) Error() {
	atomic.AddInt64(&m.errors, 1)
}

func (m *monitorFunc) getHits() int {
	return int(atomic.LoadInt64(&m.hits))
}

func (m *monitorFunc) getMisses() int {
	return int(atomic.LoadInt64(&m.misses))
}

func (m *monitorFunc) getStales() int {
	return int(atomic.LoadInt64(&m.stales))
}

func (m *monitorFunc) getBackends() int {
	return int(atomic.LoadInt64(&m.backend))
}

func (m *monitorFunc) getErrors() int {
	return int(atomic.LoadInt64(&m.errors))
}
