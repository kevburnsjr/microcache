package microcache

// Compressor is the interface for response compressors
type Compressor interface {

	// Compress compresses a response prior to being saved in the cache
	// usually by compressing the response body
	Compress(Response) Response

	// Expand decompresses a response after being retrieved from the cache
	// usually by decompressing the response body
	Expand(Response) Response
}
