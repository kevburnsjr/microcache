package microcache

// Driver is the interface for cache drivers
type Driver interface {

	// SetRequestOpts stores request options in the request cache.
	// Requests contain request-specific cache configuration based on response headers
	SetRequestOpts(string, RequestOpts) error

	// GetRequestOpts retrieves request options from the request cache
	GetRequestOpts(string) RequestOpts

	// Set stores a response object in the response cache.
	// This contains the full response as well as an expiration date.
	Set(string, Response) error

	// Get retrieves a response object from the response cache
	Get(string) Response

	// Remove removes a response object from the response cache.
	// Required by HTTP spec to purge cached responses after successful unsafe request.
	Remove(string) error

	// GetSize returns the number of objects stored in the cache
	GetSize() int
}
