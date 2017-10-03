package microcache

// Driver is the interface for cache drivers
type Driver interface {
	SetRequestOpts(string, RequestOpts) error
	GetRequestOpts(string) RequestOpts
	Set(string, Response) error
	Get(string) Response
	Remove(string) error

	// GetSize returns the number of objects stored in the cache
	GetSize() int
}
