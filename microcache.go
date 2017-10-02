// microcache is an HTTP cache implemented as Go middleware.
// Useful for APIs which serve large numbers of identical responses.
// Especially useful in high traffic microservices to improve efficiency by
// reducing read traffic through collapsed forwarding and improve availability
// by serving stale responses should synchronous dependencies become unavailable.
package microcache

import (
	"net/http"
	"strings"
	"time"
)

type microcache struct {
	Nocache              bool
	TTL                  time.Duration
	StaleIfError         time.Duration
	StaleWhileRevalidate time.Duration
	Timeout              time.Duration
	TTLSync              bool
	HashQuery            bool
	CollapsedForwarding  bool
	Vary                 []string
	Driver               Driver
	Monitor              Monitor
	Exposed              bool
}

type Config struct {
	// Nocache prevents responses from being cached by default
	// Can be overridden by the microcache-cache and microcache-nocache response headers
	Nocache bool

	// TTL specifies a default ttl for cached responses
	// Can be overridden by the microcache-ttl response header
	// Recommended: 10s
	// Default: 0
	TTL time.Duration

	// StaleIfError specifies a default stale grace period
	// If a request fails and StaleIfError is set, the object will be served as stale
	// and the response will be re-cached for the duration of this grace period
	// Can be overridden by the microcache-ttl-stale response header
	// More Info: https://tools.ietf.org/html/rfc5861
	// Recommended: 20s
	// Default: 0
	StaleIfError time.Duration

	// StaleWhileRevalidate specifies a period during which a stale response may be
	// served immediately while the resource is fetched in the background. This can be
	// useful for ensuring consistent response times at the cost of content freshness.
	// More Info: https://tools.ietf.org/html/rfc5861
	// Recommended: 20s
	// Default: 0
	StaleWhileRevalidate time.Duration

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

	// Timeout specifies the maximum execution time for backend responses
	// Example: If the underlying handler takes more than 10s to respond,
	// the request is cancelled and the response is treated as 503
	// Recommended: 10s
	// Default: 0
	Timeout time.Duration

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
func New(o Config) microcache {
	// Defaults
	m := microcache{
		Nocache:              o.Nocache,
		TTL:                  o.TTL,
		StaleIfError:         o.StaleIfError,
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
		m.Driver = NewGcacheDriver(1e4) // default 10k cache items
	}
	return m
}

// Middleware can be used to wrap an HTTP handler with microcache functionality.
// It can also be passed to http middleware providers like alice as a constructor.
//
//   mx := microcache.New(microcache.Config{TTL: 10 * time.Second})
//   newHandler := mx.Middleware(yourHandler)
//
// Or with alice
//
//  chain.Append(mx.Middleware)
//
func (m *microcache) Middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Websocket passthrough
		upgrade := strings.ToLower(r.Header.Get("connection")) == "upgrade"
		if upgrade || m.Driver == nil {
			h.ServeHTTP(w, r)
			return
		}

		// Fetch request options
		reqHash := getRequestHash(m, r)
		req := m.Driver.GetRequestOpts(reqHash)

		// Fetch cached response object
		var objHash string
		var obj Response
		if req.found {
			objHash = req.getObjectHash(reqHash, r)
			obj = m.Driver.Get(objHash)
		}

		// Non-cacheable request method passthrough and purge
		if r.Method != "GET" {
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
			if m.Exposed {
				w.Header().Set("microcache", "STALE")
			}
			go m.handleBackendResponse(h, w, r, req, reqHash, objHash, obj, true)
		} else {
			m.handleBackendResponse(h, w, r, req, reqHash, objHash, obj, false)
		}
	})
}

func (m *microcache) handleBackendResponse(
	h http.Handler,
	w http.ResponseWriter,
	r *http.Request,
	req RequestOpts,
	reqHash string,
	objHash string,
	obj Response,
	validatingStale bool,
) {
	// Backend Response
	beres := Response{header: http.Header{}}

	if req.collapsedForwarding && req.found && req.ttl > 0 {
		// forward collapse
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
		// Extend stale response expiration by staleIfError grace period
		if req.found && req.staleIfError > 0 {
			obj.setExpires(req.staleIfError, req.ttlSync)
			m.Driver.Set(objHash, obj)
		}
		if m.Monitor != nil {
			m.Monitor.Error()
		}
		if !validatingStale {
			if m.Exposed {
				w.Header().Set("microcache", "STALE")
			}
			obj.sendResponse(w)
		}
		return
	}

	if beres.status >= 200 && beres.status < 400 {
		if !req.found {
			// Store request options
			req = buildRequestOpts(m, beres, r)
			if !req.nocache {
				m.Driver.SetRequestOpts(reqHash, req)
				objHash = req.getObjectHash(reqHash, r)
			}
		}
		// Cache response
		if !req.nocache {
			beres.found = true
			beres.setExpires(req.ttl, req.ttlSync)
			m.Driver.Set(objHash, beres)
		}
	}

	if !validatingStale {
		if m.Exposed {
			w.Header().Set("microcache", "MISS")
		}
		beres.sendResponse(w)
	}
}
