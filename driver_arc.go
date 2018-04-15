package microcache

import (
	"github.com/hashicorp/golang-lru"
)

// DriverARC is a driver implementation using github.com/hashicorp/golang-lru
// ARCCache is a thread-safe fixed size Adaptive Replacement Cache (ARC).
// It requires more ram and cpu than straight LRU but can be more efficient
// https://godoc.org/github.com/hashicorp/golang-lru#ARCCache
type DriverARC struct {
	RequestCache  *lru.ARCCache
	ResponseCache *lru.ARCCache
}

// NewDriverARC returns an ARC driver.
// size determines the number of items in the cache.
// Memory usage should be considered when choosing the appropriate cache size.
// The amount of memory consumed by the driver will depend upon the response size.
// Roughly, memory = cacheSize * averageResponseSize / compression ratio
// ARC caches have additional CPU and memory overhead when compared with LRU
// ARC does not support eviction monitoring
func NewDriverARC(size int) DriverARC {
	// golang-lru segfaults when size is zero
	if size < 1 {
		size = 1
	}
	reqCache, _ := lru.NewARC(size)
	resCache, _ := lru.NewARC(size)
	return DriverARC{
		reqCache,
		resCache,
	}
}

func (c DriverARC) SetRequestOpts(hash string, req RequestOpts) error {
	c.RequestCache.Add(hash, req)
	return nil
}

func (c DriverARC) GetRequestOpts(hash string) (req RequestOpts) {
	obj, success := c.RequestCache.Get(hash)
	if success {
		req = obj.(RequestOpts)
	}
	return req
}

func (c DriverARC) Set(hash string, res Response) error {
	c.ResponseCache.Add(hash, res)
	return nil
}

func (c DriverARC) Get(hash string) (res Response) {
	obj, success := c.ResponseCache.Get(hash)
	if success {
		res = obj.(Response)
	}
	return res
}

func (c DriverARC) Remove(hash string) error {
	c.ResponseCache.Remove(hash)
	return nil
}

func (c DriverARC) GetSize() int {
	return c.ResponseCache.Len()
}
