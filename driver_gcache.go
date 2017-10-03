package microcache

import (
	"errors"
	"github.com/bluele/gcache"
)

// DriverGcache is a driver implementation based on github.com/bluele/gcache
// Example: driver := microcache.DriverGcache{
//              RequestCache:  gcache.New(5000).LRU().Build(),
//              ResponseCache: gcache.New(5000).LRU().Build(),
//          }
type DriverGcache struct {
	RequestCache  gcache.Cache
	ResponseCache gcache.Cache
}

// NewDriverGcache returns the default LFU gcache driver configuration.
// size determines the number of items in the cache.
// Memory usage should be considered when choosing the appropriate cache size.
// The amount of memory consumed by the driver will depend upon the response size.
// Roughly, memory = cacheSize * averageResponseSize
func NewDriverGcache(size int) DriverGcache {
	return DriverGcache{
		gcache.New(size).LFU().Build(),
		gcache.New(size).LFU().Build(),
	}
}

func (c DriverGcache) SetRequestOpts(hash string, req RequestOpts) error {
	return c.RequestCache.Set(hash, req)
}

func (c DriverGcache) GetRequestOpts(hash string) (req RequestOpts) {
	obj, err := c.RequestCache.Get(hash)
	if err == nil {
		req = obj.(RequestOpts)
	}
	return req
}

func (c DriverGcache) Set(hash string, res Response) error {
	return c.ResponseCache.Set(hash, res)
}

func (c DriverGcache) Get(hash string) (res Response) {
	obj, err := c.ResponseCache.Get(hash)
	if err == nil {
		res = obj.(Response)
	}
	return res
}

func (c DriverGcache) Remove(hash string) error {
	removed := c.ResponseCache.Remove(hash)
	if !removed {
		return errors.New("Could not remove item from cache")
	}
	return nil
}

func (c DriverGcache) GetSize() int {
	return c.ResponseCache.Len()
}
