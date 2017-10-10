package microcache

import (
	"github.com/hashicorp/golang-lru"
)

// DriverLRU is a driver implementation based on github.com/hashicorp/golang-lru
type DriverLRU struct {
	RequestCache  *lru.Cache
	ResponseCache *lru.Cache
}

// NewDriverLRU returns the default LRU driver configuration.
// size determines the number of items in the cache.
// Memory usage should be considered when choosing the appropriate cache size.
// The amount of memory consumed by the driver will depend upon the response size.
// Roughly, memory = cacheSize * averageResponseSize
func NewDriverLRU(size int) DriverLRU {
	reqCache, _ := lru.New(size)
	resCache, _ := lru.New(size)
	return DriverLRU{
		reqCache,
		resCache,
	}
}

func (c DriverLRU) SetRequestOpts(hash string, req RequestOpts) error {
	c.RequestCache.Add(hash, req)
	return nil
}

func (c DriverLRU) GetRequestOpts(hash string) (req RequestOpts) {
	obj, success := c.RequestCache.Get(hash)
	if success {
		req = obj.(RequestOpts)
	}
	return req
}

func (c DriverLRU) Set(hash string, res Response) error {
	c.ResponseCache.Add(hash, res)
	return nil
}

func (c DriverLRU) Get(hash string) (res Response) {
	obj, success := c.ResponseCache.Get(hash)
	if success {
		res = obj.(Response)
	}
	return res
}

func (c DriverLRU) Remove(hash string) error {
	c.ResponseCache.Remove(hash)
	return nil
}

func (c DriverLRU) GetSize() int {
	return c.ResponseCache.Len()
}
