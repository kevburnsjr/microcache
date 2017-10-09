package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
)

// This script generates a file containing 5,000 random URLs for testing
//     siege -f /tmp/urls.txt -c50 -b
func main() {
	file := ""
	for i := 0; i < 5000; i++ {
		file = file + fmt.Sprintf("http://localhost/%d\n", rand.Int())
	}
	ioutil.WriteFile("/tmp/urls.txt", []byte(file), 0644)
}
