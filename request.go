package microcache

import (
	"crypto/sha1"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func getRequestHash(m *microcache, r *http.Request) string {
	h := sha1.New()
	h.Write([]byte(r.URL.Path))
	for _, header := range m.Vary {
		h.Write([]byte("&" + header + ":" + r.Header.Get(header)))
	}
	if m.HashQuery {
		if m.QueryIgnore != nil {
			for key, values := range r.URL.Query() {
				if _, ok := m.QueryIgnore[key]; ok {
					continue
				}
				for _, value := range values {
					h.Write([]byte("&" + key + "=" + value))
				}
			}
		} else {
			h.Write([]byte(r.URL.RawQuery))
		}
	}
	return string(h.Sum(nil))
}

// RequestOpts stores per-request cache options. This is necessary to allow
// custom response headers to be evaluated, cached and applied prior to
// response object retrieval (ie. microcache-vary, microcache-nocache, etc)
type RequestOpts struct {
	found                bool
	ttl                  time.Duration
	staleIfError         time.Duration
	staleRecache         bool
	staleWhileRevalidate time.Duration
	collapsedForwarding  bool
	vary                 []string
	varyQuery            []string
	nocache              bool
}

func (req *RequestOpts) getObjectHash(reqHash string, r *http.Request) string {
	h := sha1.New()
	h.Write([]byte(reqHash))
	for _, header := range req.vary {
		h.Write([]byte("&" + header + ":" + r.Header.Get(header)))
	}
	if len(req.varyQuery) > 0 {
		queryParams := r.URL.Query()
		for _, param := range req.varyQuery {
			if vals, ok := queryParams[param]; ok {
				for _, val := range vals {
					h.Write([]byte("&" + param + "=" + val))
				}
			}
		}
	}
	return string(h.Sum(nil))
}

func buildRequestOpts(m *microcache, res Response, r *http.Request) RequestOpts {
	headers := res.header
	req := RequestOpts{
		found:                true,
		nocache:              m.Nocache,
		ttl:                  m.TTL,
		staleIfError:         m.StaleIfError,
		staleRecache:         m.StaleRecache,
		staleWhileRevalidate: m.StaleWhileRevalidate,
		collapsedForwarding:  m.CollapsedForwarding,
		vary:                 m.Vary,
	}

	// w.Header().Set("microcache-cache", "1")
	if headers.Get("microcache-cache") != "" {
		req.nocache = false
	}

	// w.Header().Set("microcache-nocache", "1")
	if headers.Get("microcache-nocache") != "" {
		req.nocache = true
	}

	// w.Header().Set("microcache-ttl", "10") // 10 seconds
	ttlHdr, _ := strconv.Atoi(headers.Get("microcache-ttl"))
	if ttlHdr > 0 {
		req.ttl = time.Duration(ttlHdr) * time.Second
	}

	// w.Header().Set("microcache-stale-if-error", "20") // 20 seconds
	staleIfErrorHdr, _ := strconv.Atoi(headers.Get("microcache-stale-if-error"))
	if staleIfErrorHdr > 0 {
		req.staleIfError = time.Duration(staleIfErrorHdr) * time.Second
	}

	// w.Header().Set("microcache-stale-while-revalidate", "20") // 20 seconds
	staleWhileRevalidateHdr, _ := strconv.Atoi(headers.Get("microcache-stale-while-revalidate"))
	if staleWhileRevalidateHdr > 0 {
		req.staleWhileRevalidate = time.Duration(staleWhileRevalidateHdr) * time.Second
	}

	// w.Header().Set("microcache-collapsed-forwarding", "1")
	if headers.Get("microcache-collapsed-forwarding") != "" {
		req.collapsedForwarding = true
	}

	// w.Header().Set("microcache-no-collapsed-forwarding", "1")
	if headers.Get("microcache-no-collapsed-forwarding") != "" {
		req.collapsedForwarding = false
	}

	// w.Header().Set("microcache-stale-recache", "1")
	if headers.Get("microcache-stale-recache") != "" {
		req.staleRecache = true
	}

	// w.Header().Set("microcache-no-stale-recache", "1")
	if headers.Get("microcache-no-stale-recache") != "" {
		req.staleRecache = false
	}

	// w.Header().Add("microcache-vary-query", "q, page, limit")
	if varyQueries, ok := headers["Microcache-Vary-Query"]; ok {
		for _, hdr := range varyQueries {
			varyQueryParams := strings.Split(hdr, ",")
			for i, v := range varyQueryParams {
				varyQueryParams[i] = strings.Trim(v, " ")
			}
			req.varyQuery = append(req.varyQuery, varyQueryParams...)
		}
	}

	// w.Header().Add("microcache-vary", "accept-language, accept-encoding")
	if varyHdr, ok := headers["Microcache-Vary"]; ok {
		for _, hdr := range varyHdr {
			varyHdrs := strings.Split(hdr, ",")
			for i, v := range varyHdrs {
				varyHdrs[i] = strings.Trim(v, " ")
			}
			req.vary = append(req.vary, varyHdrs...)
		}
	}

	// w.Header().Add("Vary", "accept-language, accept-encoding")
	if varyHdr, ok := headers["Vary"]; ok {
		for _, hdr := range varyHdr {
			varyHdrs := strings.Split(hdr, ",")
			for i, v := range varyHdrs {
				varyHdrs[i] = strings.Trim(v, " ")
			}
			req.vary = append(req.vary, varyHdrs...)
		}
	}

	return req
}
