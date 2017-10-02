package microcache

import (
	"sync"
	"time"
)

// MonitorFunc turns a function into a Monitor
func MonitorFunc(interval time.Duration, logFunc func(int, int, float64, float64, float64)) monitorFunc {
	return monitorFunc{
		interval: interval,
		logFunc:  logFunc,
	}
}

type monitorFunc struct {
	interval   time.Duration
	logFunc    func(objectCount int, size int, hitRate float64, missRate float64, errorRate float64)
	hits       int
	hitMutex   sync.Mutex
	misses     int
	missMutex  sync.Mutex
	stale      int
	staleMutex sync.Mutex
	errors     int
	errorMutex sync.Mutex
}

func (m *monitorFunc) GetInterval() time.Duration {
	return m.interval
}

func (m *monitorFunc) Log(objectCount int, size int) {
	total := m.hits + m.misses
	m.logFunc(objectCount, size, float64(m.hits/total), float64(m.misses/total), float64(m.errors/total))
}

func (m *monitorFunc) Hit() {
	m.hitMutex.Lock()
	m.hits = m.hits + 1
	m.hitMutex.Unlock()
}

func (m *monitorFunc) Miss() {
	m.missMutex.Lock()
	m.misses = m.misses + 1
	m.missMutex.Unlock()
}

func (m *monitorFunc) Stale() {
	m.staleMutex.Lock()
	m.stale = m.stale + 1
	m.staleMutex.Unlock()
}

func (m *monitorFunc) Error() {
	m.errorMutex.Lock()
	m.errors = m.errors + 1
	m.errorMutex.Unlock()
}
