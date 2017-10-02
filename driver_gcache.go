package microcache

import (
	"errors"
	"github.com/bluele/gcache"
)

// GcacheDriver is a driver implementation based on github.com/bluele/gcache
// Example: driver := microcache.GcacheDriver{
//              RequestCache:  gcache.New(5000).LRU().Build(),
//              ResponseCache: gcache.New(5000).LRU().Build(),
//          }
type GcacheDriver struct {
	RequestCache  gcache.Cache
	ResponseCache gcache.Cache
}

// NewGcacheDriver is the default LFU gcache driver.
// size determines the number of items in the cache.
// Memory usage should be considered when choosing the appropriate cache size.
// The amount of memory consumed by the driver will depend upon the response size.
// Roughly, memory = cacheSize * averageResponseSize
func NewGcacheDriver(size int) GcacheDriver {
	return GcacheDriver{
		gcache.New(size).LFU().Build(),
		gcache.New(size).LFU().Build(),
	}
}

// SetRequestOpts stores request options in the request cache.
// Requests contain request-specific cache configuration based on response headers
func (c GcacheDriver) SetRequestOpts(hash string, req RequestOpts) error {
	return c.RequestCache.Set(hash, req)
}

// GetRequestOpts retrieves request options from the request cache
func (c GcacheDriver) GetRequestOpts(hash string) (req RequestOpts) {
	obj, err := c.RequestCache.Get(hash)
	if err == nil {
		req = obj.(RequestOpts)
	}
	return req
}

// Set stores a response object in the response cache.
// This contains the full response as well as an expiration date.
func (c GcacheDriver) Set(hash string, res Response) error {
	return c.ResponseCache.Set(hash, res)
}

// Get retrieves a response object from the response cache
func (c GcacheDriver) Get(hash string) (res Response) {
	obj, err := c.ResponseCache.Get(hash)
	if err == nil {
		res = obj.(Response)
	}
	return res
}

// Remove removes a response object from the response cache.
// Required by HTTP spec to purge cached responses after successful unsafe request.
func (c GcacheDriver) Remove(hash string) error {
	removed := c.ResponseCache.Remove(hash)
	if !removed {
		return errors.New("Could not remove item from cache")
	}
	return nil
}
