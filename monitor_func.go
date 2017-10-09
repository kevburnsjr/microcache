package microcache

import (
	"sync"
	"time"
)

// MonitorFunc turns a function into a Monitor
func MonitorFunc(interval time.Duration, logFunc func(Stats)) Monitor {
	return &monitorFunc{
		interval: interval,
		logFunc:  logFunc,
	}
}

type monitorFunc struct {
	interval     time.Duration
	logFunc      func(Stats)
	hits         int
	hitMutex     sync.Mutex
	misses       int
	missMutex    sync.Mutex
	stales       int
	staleMutex   sync.Mutex
	backend      int
	backendMutex sync.Mutex
	errors       int
	errorMutex   sync.Mutex
	stop         chan bool
}

func (m *monitorFunc) GetInterval() time.Duration {
	return m.interval
}

func (m *monitorFunc) Log(stats Stats) {
	// hits
	m.hitMutex.Lock()
	stats.Hits = m.hits
	m.hits = 0
	m.hitMutex.Unlock()

	// misses
	m.missMutex.Lock()
	stats.Misses = m.misses
	m.misses = 0
	m.missMutex.Unlock()

	// stales
	m.staleMutex.Lock()
	stats.Stales = m.stales
	m.stales = 0
	m.staleMutex.Unlock()

	// backend
	m.backendMutex.Lock()
	stats.Backend = m.backend
	m.backend = 0
	m.backendMutex.Unlock()

	// errors
	m.errorMutex.Lock()
	stats.Errors = m.errors
	m.errors = 0
	m.errorMutex.Unlock()

	// log
	m.logFunc(stats)
}

func (m *monitorFunc) Hit() {
	m.hitMutex.Lock()
	m.hits += 1
	m.hitMutex.Unlock()
}

func (m *monitorFunc) Miss() {
	m.missMutex.Lock()
	m.misses += 1
	m.missMutex.Unlock()
}

func (m *monitorFunc) Stale() {
	m.staleMutex.Lock()
	m.stales += 1
	m.staleMutex.Unlock()
}

func (m *monitorFunc) Backend() {
	m.backendMutex.Lock()
	m.backend += 1
	m.backendMutex.Unlock()
}

func (m *monitorFunc) Error() {
	m.errorMutex.Lock()
	m.errors += 1
	m.errorMutex.Unlock()
}
