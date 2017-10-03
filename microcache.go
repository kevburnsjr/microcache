// microcache is a non-standard HTTP microcache implemented as Go middleware.
package microcache

import (
	"net/http"
	"strings"
	"time"
)

type Microcache interface {
	Middleware(http.Handler) http.Handler
	Start()
	Stop()
}

type microcache struct {
	Nocache              bool
	Timeout              time.Duration
	TTL                  time.Duration
	StaleIfError         time.Duration
	StaleRecache         bool
	StaleWhileRevalidate time.Duration
	TTLSync              bool
	HashQuery            bool
	CollapsedForwarding  bool
	Vary                 []string
	Driver               Driver
	Monitor              Monitor
	Exposed              bool

	stopMonitor chan bool
}

type Config struct {
	// Nocache prevents responses from being cached by default
	// Can be overridden by the microcache-cache and microcache-nocache response headers
	Nocache bool

	// Timeout specifies the maximum execution time for backend responses
	// Example: If the underlying handler takes more than 10s to respond,
	// the request is cancelled and the response is treated as 503
	// Recommended: 10s
	// Default: 0
	Timeout time.Duration

	// TTL specifies a default ttl for cached responses
	// Can be overridden by the microcache-ttl response header
	// Recommended: 10s
	// Default: 0
	TTL time.Duration

	// StaleWhileRevalidate specifies a period during which a stale response may be
	// served immediately while the resource is fetched in the background. This can be
	// useful for ensuring consistent response times at the cost of content freshness.
	// More Info: https://tools.ietf.org/html/rfc5861
	// Recommended: 20s
	// Default: 0
	StaleWhileRevalidate time.Duration

	// StaleIfError specifies a default stale grace period
	// If a request fails and StaleIfError is set, the object will be served as stale
	// and the response will be re-cached for the duration of this grace period
	// Can be overridden by the microcache-ttl-stale response header
	// More Info: https://tools.ietf.org/html/rfc5861
	// Recommended: 20s
	// Default: 0
	StaleIfError time.Duration

	// StaleRecache specifies whether to re-cache the response object for ttl while serving
	// stale response on backend error
	// Recommended: true
	// Default: false
	StaleRecache bool

	// StaleWhileRevalidate specifies a period during which a stale response may be
	// served immediately while the resource is fetched in the background. This can be
	// useful for ensuring consistent response times at the cost of content freshness.
	// Recommended: 20s
	// Default: 0
	CollapsedFowarding bool

	// TTLSync will lock TTLs to the system clock to improve consistency between caches.
	// This can help prevent parallel caches from expiring response objects at different times.
	// This assumes that the parallel caches in question have synchronised clocks (see ntpd)
	// This will cause response expiration to vary between now and now + ttl due to rounding.
	// A 10s TTL applied at 09:12:43 will set expires to 09:12:50 rather than 09:12:53 (7s).
	// A 10s TTL applied at 09:12:49 will set expires to 09:12:50 rather than 09:12:59 (1s).
	// This should only be set where consistency is more important than efficiency.
	// Can be overridden by the microcache-ttl-sync response header
	// Default: false
	TTLSync bool

	// HashQuery determines whether all query parameters in the request URI
	// should be hashed to differentiate requests
	// Default: false
	HashQuery bool

	// Vary specifies a list of http request headers by which all requests
	// should be differentiated. When making use of this option, it may be a good idea
	// to normalize these headers first using a separate piece of middleware.
	//
	//   []string{"accept-language", "accept-encoding", "xml-http-request"}
	//
	// Default: []string{}
	Vary []string

	// Driver specifies a cache storage driver.
	// Default: gcache with 10,000 item capacity
	Driver Driver

	// Monitor is an optional parameter which will periodically report statistics about
	// the cache to enable monitoring of cache size, cache efficiency and error rate
	// Default: nil
	Monitor Monitor

	// Exposed determines whether to add a header to the response indicating the response state
	// Microcache: ( HIT | MISS | STALE )
	// Default: 0
	Exposed bool
}

// New creates and returns a configured microcache instance
func New(o Config) Microcache {
	// Defaults
	m := microcache{
		Nocache:              o.Nocache,
		TTL:                  o.TTL,
		StaleIfError:         o.StaleIfError,
		StaleRecache:         o.StaleRecache,
		StaleWhileRevalidate: o.StaleWhileRevalidate,
		TTLSync:              o.TTLSync,
		Timeout:              o.Timeout,
		HashQuery:            o.HashQuery,
		CollapsedForwarding:  true,
		Vary:                 o.Vary,
		Driver:               o.Driver,
		Monitor:              o.Monitor,
		Exposed:              o.Exposed,
	}
	if o.Driver == nil {
		m.Driver = NewDriverGcache(1e4) // default 10k cache items
	}
	m.Start()
	return &m
}

// Middleware can be used to wrap an HTTP handler with microcache functionality.
// It can also be passed to http middleware providers like alice as a constructor.
//
//     mx := microcache.New(microcache.Config{TTL: 10 * time.Second})
//     newHandler := mx.Middleware(yourHandler)
//
// Or with alice
//
//    chain.Append(mx.Middleware)
//
func (m *microcache) Middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Websocket passthrough
		upgrade := strings.ToLower(r.Header.Get("connection")) == "upgrade"
		if upgrade || m.Driver == nil {
			if m.Monitor != nil {
				m.Monitor.Miss()
			}
			h.ServeHTTP(w, r)
			return
		}

		// Fetch request options
		reqHash := getRequestHash(m, r)
		req := m.Driver.GetRequestOpts(reqHash)

		// Hard passthrough on non cacheable requests
		if req.nocache {
			if m.Monitor != nil {
				m.Monitor.Miss()
			}
			h.ServeHTTP(w, r)
			return
		}

		// Fetch cached response object
		var objHash string
		var obj Response
		if req.found {
			objHash = req.getObjectHash(reqHash, r)
			obj = m.Driver.Get(objHash)
		}

		// Non-cacheable request method passthrough and purge
		if r.Method != "GET" && r.Method != "HEAD" {
			if m.Monitor != nil {
				m.Monitor.Miss()
			}
			if obj.found {
				// HTTP spec requires caches to purge cached responses following
				//  successful unsafe request.
				ptw := passthroughWriter{w, 0}
				h.ServeHTTP(ptw, r)
				if ptw.status >= 200 && ptw.status < 400 {
					m.Driver.Remove(objHash)
				}
			} else {
				h.ServeHTTP(w, r)
			}
			return
		}

		// Fresh response object found
		if obj.found && obj.expires.After(time.Now()) {
			if m.Monitor != nil {
				m.Monitor.Hit()
			}
			if m.Exposed {
				w.Header().Set("microcache", "HIT")
			}
			obj.sendResponse(w)
			return
		}

		// Stale While Revalidate
		if obj.found && req.staleWhileRevalidate > 0 &&
			obj.expires.Add(req.staleWhileRevalidate).After(time.Now()) {
			obj.sendResponse(w)
			if m.Monitor != nil {
				m.Monitor.Stale()
			}
			if m.Exposed {
				w.Header().Set("microcache", "STALE")
			}
			go m.handleBackendResponse(h, w, r, reqHash, req, objHash, obj, true)
			return
		} else {
			m.handleBackendResponse(h, w, r, reqHash, req, objHash, obj, false)
			return
		}
	})
}

func (m *microcache) handleBackendResponse(
	h http.Handler,
	w http.ResponseWriter,
	r *http.Request,
	reqHash string,
	req RequestOpts,
	objHash string,
	obj Response,
	revalidating bool,
) {
	// Backend Response
	beres := Response{header: http.Header{}}

	if req.found && req.collapsedForwarding && req.ttl > 0 {
		// collapsedForwarding not yet implemented
		// probably requires a threadsafe map[reqHash]sync.Mutex
		// may need to extract more logic from Middleware func
		// would rather implement this after testing is in place
	}

	// Execute request
	if m.Timeout > 0 {
		th := http.TimeoutHandler(h, m.Timeout, "Timed out")
		th.ServeHTTP(&beres, r)
	} else {
		h.ServeHTTP(&beres, r)
	}

	// Serve Stale
	if beres.status >= 500 && obj.found {
		serveStale := obj.expires.Add(req.staleIfError).After(time.Now())
		// Extend stale response expiration by staleIfError grace period
		if req.found && serveStale && req.staleRecache {
			obj.setExpires(req.ttl, req.ttlSync)
			m.Driver.Set(objHash, obj)
		}
		if m.Monitor != nil {
			m.Monitor.Error()
		}
		if !revalidating && serveStale {
			if m.Monitor != nil {
				m.Monitor.Stale()
			}
			if m.Exposed {
				w.Header().Set("microcache", "STALE")
			}
			obj.sendResponse(w)
			return
		}
	}

	if beres.status >= 200 && beres.status < 400 {
		if !req.found {
			// Store request options
			req = buildRequestOpts(m, beres, r)
			m.Driver.SetRequestOpts(reqHash, req)
			objHash = req.getObjectHash(reqHash, r)
		}
		// Cache response
		if !req.nocache {
			beres.found = true
			beres.setExpires(req.ttl, req.ttlSync)
			m.Driver.Set(objHash, beres)
		}
	}

	if !revalidating {
		if m.Monitor != nil {
			m.Monitor.Miss()
		}
		if m.Exposed {
			w.Header().Set("microcache", "MISS")
		}
		beres.sendResponse(w)
		return
	}
}

// Start starts the monitor and any other required background processes
func (m *microcache) Start() {
	m.stopMonitor = make(chan bool)
	if m.Monitor != nil {
		go func() {
			for {
				select {
				case <-time.After(m.Monitor.GetInterval()):
					m.Monitor.Log(Stats{
						Size: m.Driver.GetSize(),
					})
				case <-m.stopMonitor:
					return
				}
			}
		}()
	}
}

// Stop stops the monitor and any other required background processes
func (m *microcache) Stop() {
	m.stopMonitor <- true
}
