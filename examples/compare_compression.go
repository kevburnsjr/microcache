package main

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/golang/snappy"
)

func main() {

	var filepath *string = flag.String("f", "", "File to compress")
	flag.Parse()

	if *filepath == "" {
		fmt.Println("Error: Missing Flag -f filepath (required)")
		return
	}

	// Create a file (json or otherwise) using your own data to see how it
	// compresses with these different compression algorithms.
	dat, _ := ioutil.ReadFile(*filepath)

	orig := len(dat)

	fmt.Printf("Original: %d bytes\n", orig)

	var c []byte

	start := time.Now()
	for i := 0; i < 1e2; i++ {
		c = compressZlib(dat)
	}
	fmt.Printf("zlib compress %v %d bytes (%.2fx)\n", time.Since(start), len(c), float64(len(c))/float64(orig))

	start = time.Now()
	for i := 0; i < 1e2; i++ {
		expandZlib(c)
	}
	fmt.Printf("zlib expand %v\n", time.Since(start))

	start = time.Now()
	for i := 0; i < 1e2; i++ {
		c = compressGzip(dat)
	}
	fmt.Printf("gzip compress %v %d bytes (%.2fx)\n", time.Since(start), len(c), float64(len(c))/float64(orig))

	start = time.Now()
	for i := 0; i < 1e2; i++ {
		expandGzip(c)
	}
	fmt.Printf("gzip expand %v\n", time.Since(start))

	start = time.Now()
	for i := 0; i < 1e2; i++ {
		c = compressSnappy(dat)
	}
	fmt.Printf("snappy compress %v %d bytes (%.2fx)\n", time.Since(start), len(c), float64(len(c))/float64(orig))

	start = time.Now()
	for i := 0; i < 1e2; i++ {
		expandSnappy(c)
	}
	fmt.Printf("snappy expand %v\n", time.Since(start))

}

func compressZlib(in []byte) []byte {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(in)
	w.Close()
	return b.Bytes()
}

func expandZlib(in []byte) []byte {
	buf := bytes.NewBuffer(in)
	r, _ := zlib.NewReader(buf)
	out, _ := ioutil.ReadAll(r)
	r.Close()
	return out
}

func compressGzip(in []byte) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write(in)
	w.Close()
	return b.Bytes()
}

func expandGzip(in []byte) []byte {
	buf := bytes.NewBuffer(in)
	r, _ := gzip.NewReader(buf)
	out, _ := ioutil.ReadAll(r)
	r.Close()
	return out
}

func compressSnappy(in []byte) []byte {
	return snappy.Encode(nil, in)
}

func expandSnappy(in []byte) []byte {
	out, _ := snappy.Decode(nil, in)
	return out
}
