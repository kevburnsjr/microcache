package microcache

import (
	"github.com/golang/snappy"
)

// CompressorSnappy is a Snappy compressor
// 14x faster compress than gzip
// 8x faster expand than gzip
// ~ 1.5 - 2x larger result (see README)
type CompressorSnappy struct {
}

func (c CompressorSnappy) Compress(res Response) Response {
	newres := res.clone()
	newres.body = snappy.Encode(nil, res.body)
	return newres
}

func (c CompressorSnappy) Expand(res Response) Response {
	res.body, _ = snappy.Decode(nil, res.body)
	return res
}
