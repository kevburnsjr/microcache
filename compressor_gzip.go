package microcache

import (
	"compress/gzip"
	"io/ioutil"
	"bytes"
)

// CompressorGzip is a gzip compressor
type CompressorGzip struct {
}

// Compress clones the supplied object
// This causes a memcopy of the response body, slowing down cache writes
// Could probably be optimized
func (c CompressorGzip) Compress(res Response) Response {
	newres := res.clone()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write(res.body)
	zw.Close()
	newres.body = buf.Bytes()
	return newres
}

// Expand modifies the supplied object
// Does not require memcopy, performance on par with no compression
func (c CompressorGzip) Expand(res Response) Response {
	buf := bytes.NewBuffer(res.body)
	zr, _ := gzip.NewReader(buf)
	res.body, _ = ioutil.ReadAll(zr)
	zr.Close()
	return res
}
