package microcache

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

// CompressorGzip is a gzip compressor
type CompressorGzip struct {
}

func (c CompressorGzip) Compress(res Response) Response {
	newres := res.clone()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write(res.body)
	zw.Close()
	newres.body = buf.Bytes()
	return newres
}

func (c CompressorGzip) Expand(res Response) Response {
	buf := bytes.NewBuffer(res.body)
	zr, _ := gzip.NewReader(buf)
	res.body, _ = ioutil.ReadAll(zr)
	zr.Close()
	return res
}
