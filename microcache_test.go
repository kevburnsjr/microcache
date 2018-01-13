package microcache

import (
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
	if testMonitor.misses != 2 || testMonitor.hits != 2 {
		t.Log("TTL not respected - got", testMonitor.hits, "hits")
		t.Fail()
	}
	cache.Stop()
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
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
		"/?a=1",
	})
	if testMonitor.misses != 2 {
		t.Log("HashQuery not respected - got", testMonitor.misses, "misses")
		t.Fail()
	}
	cache.Stop()
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
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
		"/?a=1",
	})
	if testMonitor.misses != 1 {
		t.Log("HashQuery not ignored - got", testMonitor.misses, "misses")
		t.Fail()
	}
	cache.Stop()
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
		Driver:      NewDriverLRU(10),
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
	cache.Start()
	handler := cache.Middleware(http.HandlerFunc(failureHandler))

	// prime cache
	batchGet(handler, []string{
		"/",
		"/",
	})
	if testMonitor.misses != 1 || testMonitor.hits != 1 {
		t.Log("StaleIfError not respected - got", testMonitor.misses, "misses")
		t.Fail()
	}

	// stale after 30s
	cache.offsetIncr(30 * time.Second)
	batchGet(handler, []string{
		"/?fail=1",
	})
	if testMonitor.stales != 1 {
		t.Log("StaleIfError not respected - got", testMonitor.stales, "stales")
		t.Fail()
	}

	// error after 600s
	cache.offsetIncr(600 * time.Second)
	batchGet(handler, []string{
		"/?fail=1",
	})
	if testMonitor.errors != 2 || testMonitor.stales != 1 {
		t.Log("StaleIfError not respected - got", testMonitor.errors, "errors")
		t.Fail()
	}
	cache.Stop()
}

// StaleWhilRevalidate
func TestStaleWhilRevalidate(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:                  30 * time.Second,
		StaleWhileRevalidate: 30 * time.Second,
		Monitor:              testMonitor,
		Driver:               NewDriverLRU(10),
	})
	cache.Start()
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
	cache.Start()
	handler := cache.Middleware(http.HandlerFunc(failureHandler))

	// prime cache
	batchGet(handler, []string{
		"/",
		"/",
	})
	if testMonitor.misses != 1 || testMonitor.hits != 1 {
		t.Log("StaleRecache not respected - got", testMonitor.misses, "misses")
		t.Fail()
	}

	// stale after 30s
	cache.offsetIncr(30 * time.Second)
	batchGet(handler, []string{
		"/?fail=1",
	})
	if testMonitor.stales != 1 {
		t.Log("StaleIfError not respected - got", testMonitor.stales, "stales")
		t.Fail()
	}

	// hit when stale is recached
	batchGet(handler, []string{
		"/?fail=1",
	})
	if testMonitor.hits != 2 {
		t.Log("StaleRecache not respected - got", testMonitor.errors, "errors")
		t.Fail()
	}
	cache.Stop()
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
	handler := cache.Middleware(http.HandlerFunc(slowSuccessHandler))
	start := time.Now()
	batchGet(handler, []string{
		"/",
	})
	if testMonitor.errors != 1 || time.Since(start) > 20*time.Millisecond {
		t.Log("Timeout not respected - got", testMonitor.errors, "errors")
		t.Fail()
	}
	cache.Stop()
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
	if testMonitor.misses != 1 || testMonitor.hits != 5 || time.Since(start) > 20*time.Millisecond {
		t.Log("CollapsedFowarding not respected - got", testMonitor.hits, "hits")
		t.Fail()
	}
	cache.Stop()
}

// SuppressAgeHeader
func TestAgeHeader(t *testing.T) {
	// Age header is added by default
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:               30 * time.Second,
		Monitor:           testMonitor,
		Driver:            NewDriverLRU(10),
	})
	handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
	})
	cache.offsetIncr(20 * time.Second)
	w := getResponse(handler, "/")
	if w.Header().Get("age") != "20" {
		t.Log("Age header was not correct \"", w.Header().Get("age"), "\" != 20")
		t.Fail()
	}
	cache.Stop()
	// Age header can be suppressed
	cache = New(Config{
		TTL:               30 * time.Second,
		SuppressAgeHeader: true,
		Monitor:           testMonitor,
		Driver:            NewDriverLRU(10),
	})
	handler = cache.Middleware(http.HandlerFunc(noopSuccessHandler))
	batchGet(handler, []string{
		"/",
	})
	w = getResponse(handler, "/")
	if w.Header().Get("age") != "" {
		t.Log("Age header was added when it should be empty")
		t.Fail()
	}
	cache.Stop()
}

// ARCCache should work as expected
func TestARCCache(t *testing.T) {
	testMonitor := &monitorFunc{interval: 100 * time.Second, logFunc: func(Stats) {}}
	cache := New(Config{
		TTL:     30 * time.Second,
		Monitor: testMonitor,
		Driver:  NewDriverARC(10),
	})
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
	if testMonitor.misses != 2 || testMonitor.hits != 2 {
		t.Log("TTL not respected by ARC - got", testMonitor.hits, "hits")
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
