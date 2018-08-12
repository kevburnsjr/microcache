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
	date    time.Time
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

func (res *Response) clone() Response {
	return Response{
		found:   res.found,
		date:    res.date,
		expires: res.expires,
		status:  res.status,
		header:  res.header,
		body:    res.body,
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
