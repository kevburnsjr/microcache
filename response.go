package microcache

import (
	"net/http"
	"strings"
	"time"
)

// Response is used both as a cache object for the response
// and to wrap http.ResponseWriter for downstream requests.
type Response struct {
	found   bool
	expires time.Time
	status  int
	header  http.Header
	body    []byte
}

func (res *Response) Write(b []byte) (int, error) {
	res.body = append(res.body, b...)
	return len(b), nil
}

func (res *Response) Header() http.Header {
	return res.header
}

func (res *Response) WriteHeader(code int) {
	res.status = code
}

func (res *Response) sendResponse(w http.ResponseWriter) {
	for header, values := range res.header {
		// Do not forward microcache headers to client
		if strings.HasPrefix(header, "Microcache-") {
			continue
		}
		for _, val := range values {
			w.Header().Add(header, val)
		}
	}
	w.WriteHeader(res.status)
	w.Write(res.body)
	return
}
func (res *Response) setExpires(ttl time.Duration, sync bool) {
	if sync {
		now := time.Now()
		res.expires = now.Round(ttl).Add(ttl)
		if res.expires.After(now.Add(ttl)) {
			res.expires = res.expires.Add(-1 * ttl)
		}
	} else {
		res.expires = time.Now().Add(ttl)
	}
}

type passthroughWriter struct {
	http.ResponseWriter
	status int
}

func (w passthroughWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
