package microcache

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TTL should be respected
func TestTTL(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:     30 * time.Second,
		Monitor: testMonitor,
		Driver:  NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
		"/",
	})
	cache.offsetIncr(30 * time.Second)
	batchGet(handler, []string{
		"/",
		"/",
	})
	if testMonitor.getMisses() != 2 || testMonitor.getHits() != 2 {
		t.Fatal("TTL not respected - got", testMonitor.getHits(), "hits")
	}
}

// HashQuery
func TestHashQuery(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:       30 * time.Second,
		HashQuery: true,
		Monitor:   testMonitor,
		Driver:    NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
		"/?a=1",
	})
	if testMonitor.getMisses() != 2 {
		t.Fatal("HashQuery not respected - got", testMonitor.getMisses(), "misses")
	}
}

// HashQuery Disabled
func TestHashQueryDisabled(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:       30 * time.Second,
		HashQuery: false,
		Monitor:   testMonitor,
		Driver:    NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
		"/?a=1",
	})
	if testMonitor.getMisses() != 1 {
		t.Fatal("HashQuery not ignored - got", testMonitor.getMisses(), "misses")
	}
}

// QueryIgnore should be respected when HashQuery is true
func TestQueryIgnore(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:         30 * time.Second,
		HashQuery:   true,
		QueryIgnore: []string{"a"},
		Monitor:     testMonitor,
		Driver:      NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
		"/?a=1",
		"/?a=2",
	})
	if testMonitor.getMisses() != 1 || testMonitor.getHits() != 2 {
		t.Fatal("Query parameters not ignored - got", testMonitor.getMisses(), "misses")
	}
}

// QueryIgnore should be disregarded when HashQuery is false
func TestQueryIgnoreDisabled(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:         30 * time.Second,
		HashQuery:   false,
		QueryIgnore: []string{"a"},
		Monitor:     testMonitor,
		Driver:      NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
		"/?a=1",
		"/?b=2",
	})
	if testMonitor.getMisses() != 1 {
		t.Fatal("Query parameters ignored - got", testMonitor.getMisses(), "misses")
	}
}

// StaleWhileRevalidate
func TestStaleWhileRevalidate(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:                  30 * time.Second,
		StaleWhileRevalidate: 30 * time.Second,
		Monitor:              testMonitor,
		Driver:               NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))

	// prime cache
	batchGet(handler, []string{
		"/",
		"/",
	})
	if testMonitor.getMisses() != 1 || testMonitor.getHits() != 1 {
		t.Fatal("StaleWhileRevalidate not respected - got", testMonitor.getMisses(), "misses")
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
	if testMonitor.getStales() != 1 || testMonitor.getHits() != 2 {
		t.Fatal("StaleWhileRevalidate not respected - got", testMonitor.getStales(), "stales")
	}
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
	defer cache.Stop()
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
	if testMonitor.getMisses() != 1 || testMonitor.getStales() != 6 ||
		testMonitor.getBackends() != 2 || end > 20*time.Millisecond {
		t.Logf("%#v", testMonitor)
		t.Fatal("CollapsedFowarding and StaleWhileRevalidate not respected - got", testMonitor.getBackends(), "backend")
	}
}

// StaleIfError
func TestStaleIfError(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:          30 * time.Second,
		StaleIfError: 600 * time.Second,
		Monitor:      testMonitor,
		QueryIgnore:  []string{"fail"},
		Driver:       NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(failureHandler))

	// prime cache
	batchGet(handler, []string{
		"/",
		"/",
	})
	if testMonitor.getMisses() != 1 || testMonitor.getHits() != 1 {
		t.Fatal("StaleIfError not respected - got", testMonitor.getMisses(), "misses")
	}

	// stale after 30s
	cache.offsetIncr(30 * time.Second)
	batchGet(handler, []string{
		"/?fail=1",
	})
	if testMonitor.getStales() != 1 {
		t.Fatal("StaleIfError not respected - got", testMonitor.getStales(), "stales")
	}

	// error after 600s
	cache.offsetIncr(600 * time.Second)
	batchGet(handler, []string{
		"/?fail=1",
	})
	if testMonitor.getErrors() != 2 || testMonitor.getStales() != 1 {
		t.Fatal("StaleIfError not respected - got", testMonitor.getErrors(), "errors")
	}
}

// StaleRecache
func TestStaleRecache(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:          30 * time.Second,
		StaleIfError: 600 * time.Second,
		StaleRecache: true,
		Monitor:      testMonitor,
		QueryIgnore:  []string{"fail"},
		Driver:       NewDriverLRU(10),
	})
	defer cache.Stop()

	handler := cache.Middleware(http.HandlerFunc(failureHandler))

	// prime cache
	batchGet(handler, []string{
		"/",
		"/",
	})
	if testMonitor.getMisses() != 1 || testMonitor.getHits() != 1 {
		t.Fatal("StaleRecache not respected - got", testMonitor.getMisses(), "misses")
	}

	// stale after 30s
	cache.offsetIncr(30 * time.Second)
	batchGet(handler, []string{
		"/?fail=1",
	})
	if testMonitor.getStales() != 1 {
		t.Fatal("StaleIfError not respected - got", testMonitor.getStales(), "stales")
	}

	// hit when stale is recached
	batchGet(handler, []string{
		"/?fail=1",
	})
	if testMonitor.getHits() != 2 {
		t.Fatal("StaleRecache not respected - got", testMonitor.getErrors(), "errors")
	}
}

// Timeout
func TestTimeout(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:     30 * time.Second,
		Timeout: 10 * time.Millisecond,
		Monitor: testMonitor,
		Driver:  NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(slowSuccessHandler))
	start := time.Now()
	batchGet(handler, []string{
		"/",
	})
	if testMonitor.getErrors() != 1 || time.Since(start) > 20*time.Millisecond {
		t.Fatal("Timeout not respected - got", testMonitor.getErrors(), "errors")
	}
}

// CollapsedFowarding
func TestCollapsedFowarding(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:                 30 * time.Second,
		CollapsedForwarding: true,
		Monitor:             testMonitor,
		Driver:              NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(timelySuccessHandler))
	start := time.Now()
	parallelGet(handler, []string{
		"/",
		"/",
		"/",
		"/",
		"/",
		"/",
	})
	if testMonitor.getMisses() != 1 || testMonitor.getHits() != 5 || time.Since(start) > 20*time.Millisecond {
		t.Fatal("CollapsedFowarding not respected - got", testMonitor.getHits(), "hits")
	}
}

// SuppressAgeHeader
func TestAgeHeader(t *testing.T) {
	// Age header is added by default
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:     30 * time.Second,
		Monitor: testMonitor,
		Driver:  NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
	})
	cache.offsetIncr(20 * time.Second)
	w := getResponse(handler, "/")
	if w.Header().Get("age") != "20" {
		t.Fatal("Age header was not correct \"", w.Header().Get("age"), "\" != 20")
	}
}

// SuppressAgeHeaderSuppression
func TestAgeHeaderSuppression(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:               30 * time.Second,
		SuppressAgeHeader: true,
		Monitor:           testMonitor,
		Driver:            NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
	})
	w := getResponse(handler, "/")
	if w.Header().Get("age") != "" {
		t.Fatal("Age header was added when it should be empty")
	}
}

// ARCCache should work as expected
func TestARCCache(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:     30 * time.Second,
		Monitor: testMonitor,
		Driver:  NewDriverARC(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
		"/",
	})
	cache.offsetIncr(30 * time.Second)
	batchGet(handler, []string{
		"/",
		"/",
	})
	if testMonitor.getMisses() != 2 || testMonitor.getHits() != 2 {
		t.Fatal("TTL not respected by ARC - got", testMonitor.getHits(), "hits")
	}
}

// Multiple calls to Start should not cause race conditions
func TestMultipleStart(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:     30 * time.Second,
		Monitor: testMonitor,
		Driver:  NewDriverLRU(10),
	})
	defer cache.Stop()
	cache.Start()
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
		"/",
	})
	cache.offsetIncr(30 * time.Second)
	batchGet(handler, []string{
		"/",
		"/",
	})
}

// Without WriteHeader
func TestNoWriteHeader(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:     30 * time.Second,
		Monitor: testMonitor,
		Driver:  NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	batchGet(handler, []string{
		"/",
		"/",
	})
	if testMonitor.getMisses() != 1 || testMonitor.getHits() != 1 {
		t.Fatal("WriteHeader not implicitly called", testMonitor.getHits(), "hits")
	}
}

// Stop
func TestStop(t *testing.T) {
	cache := New(Config{})
	done := make(chan bool)
	go func() {
		cache.Stop()
		done <- true
	}()
	select {
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Middleware failed to stop")
	case <-done:
		return
	}
}

// --- helper funcs ---

func batchGet(handler http.Handler, urls []string) {
	for _, url := range urls {
		r, _ := http.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		// Same check as https://github.com/golang/go/blob/d6ffc1d8394d6f6420bb92d79d320da88720fbe0/src/net/http/server.go#L1090
		if code := w.Code; code < 100 || code > 999 {
			panic(fmt.Sprintf("invalid WriteHeader code %v", code))
		}
	}
}

func parallelGet(handler http.Handler, urls []string) {
	var wg sync.WaitGroup
	for _, url := range urls {
		r1, _ := http.NewRequest("GET", url, nil)
		wg.Add(1)
		go func() {
			handler.ServeHTTP(httptest.NewRecorder(), r1)
			wg.Done()
		}()
	}
	wg.Wait()
}

func getResponse(handler http.Handler, url string) *httptest.ResponseRecorder {
	r, _ := http.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func noopSuccessHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "done", 200)
}

func failureHandler(w http.ResponseWriter, r *http.Request) {
	fail := r.FormValue("fail")
	if fail != "" {
		http.Error(w, "fail", 500)
	} else {
		http.Error(w, "done", 200)
	}
}

func slowSuccessHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(100 * time.Millisecond)
	http.Error(w, "done", 200)
}

func timelySuccessHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(10 * time.Millisecond)
	http.Error(w, "done", 200)
}
