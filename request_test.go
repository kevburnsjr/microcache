package microcache

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

// buildRequestOpts detects response headers appropriately
func TestBuildRequestOpts(t *testing.T) {
	var i = 0
	r, _ := http.NewRequest("GET", "/", nil)
	type tc struct {
		hdr string
		val string
		exp RequestOpts
	}
	var runCases = func(m *microcache, cases []tc) {
		for _, c := range cases {
			res := Response{header: http.Header{}}
			res.Header().Set(c.hdr, c.val)
			reqOpts := buildRequestOpts(m, res, r)
			reqOpts.found = false
			if !reflect.DeepEqual(reqOpts, c.exp) {
				t.Fatalf("Mismatch in case %d\n%#v\n%#v", i + 1, reqOpts, c.exp)
			}
			i++
		}
	}
	runCases(New(Config{}), []tc {
		{"microcache-nocache", "1", RequestOpts{nocache: true}},
		{"microcache-ttl", "10", RequestOpts{ttl: time.Duration(10 * time.Second)}},
		{"microcache-stale-if-error", "10", RequestOpts{staleIfError: time.Duration(10 * time.Second)}},
		{"microcache-stale-while-revalidate", "10", RequestOpts{staleWhileRevalidate: time.Duration(10 * time.Second)}},
		{"microcache-collapsed-fowarding", "1", RequestOpts{collapsedForwarding: true}},
		{"microcache-stale-recache", "1", RequestOpts{staleRecache: true}},
		{"Microcache-Vary-Query", "a", RequestOpts{varyQuery: []string{"a"}}},
	})
	runCases(New(Config{Nocache: true}), []tc {
		{"microcache-cache", "1", RequestOpts{nocache: false}},
	})
	runCases(New(Config{CollapsedForwarding: true}), []tc {
		{"microcache-no-collapsed-fowarding", "1", RequestOpts{collapsedForwarding: false}},
	})
	runCases(New(Config{StaleRecache: true}), []tc {
		{"microcache-no-stale-recache", "1", RequestOpts{staleRecache: false}},
	})
	runCases(New(Config{Vary: []string{"a"}}), []tc {
		{"Microcache-Vary", "b", RequestOpts{vary: []string{"a", "b"}}},
	})
	runCases(New(Config{Vary: []string{"a"}}), []tc {
		{"Vary", "b", RequestOpts{vary: []string{"a", "b"}}},
	})
}
