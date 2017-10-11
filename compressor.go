package microcache

// Compressor is the interface for response compressors
type Compressor interface {

	// Compress compresses a response prior to being saved in the cache and returns a clone
	// usually by compressing the response body
	Compress(Response) Response

	// Expand decompresses a response's body (destructively)
	Expand(Response) Response
}
