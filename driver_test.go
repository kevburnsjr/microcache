package microcache

import (
	"net/http"
	"testing"
)

// Remove should work as expected
func TestRemove(t *testing.T) {
	var testDriver = func(name string, d Driver) {
		cache := New(Config{Driver:  d})
		defer cache.Stop()
		handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
		batchGet(handler, []string{
			"/",
		})
		if d.GetSize() != 1 {
			t.Fatalf("%s Driver reports inaccurate length", name)
		}
		r, _ := http.NewRequest("GET", "/", nil)
		reqHash := getRequestHash(cache, r)
		reqOpts := buildRequestOpts(cache, Response{}, r)
		objHash := reqOpts.getObjectHash(reqHash, r)
		d.Remove(objHash)
		if d.GetSize() != 0 {
			t.Fatalf("%s Driver cannot delete items", name)
		}
	}
	testDriver("ARC", NewDriverARC(10))
	testDriver("LRU", NewDriverLRU(10))
}

// Empty init should not fatal
func TestEmptyInit(t *testing.T) {
	var testDriver = func(name string, d Driver) {
		cache := New(Config{Driver:  d})
		defer cache.Stop()
		handler := cache.Middleware(http.HandlerFunc(noopSuccessHandler))
		batchGet(handler, []string{
			"/a",
			"/b",
		})
		if d.GetSize() != 1 {
			t.Fatalf("%s Driver should have length 1", name)
		}
	}
	testDriver("ARC", NewDriverARC(0))
	testDriver("LRU", NewDriverLRU(0))
}
