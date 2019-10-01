package microcache

import (
	"bytes"
	"testing"
)

var zipTest = []byte(`{"firstName":"John","lastName":"Smith","isAlive":true,"age":27,"address":{"streetAddress":"21 2nd Street","city":"New York","state":"NY","postalCode":"10021-3100"},"phoneNumbers":[{"type":"home","number":"212 555-1234"},{"type":"office","number":"646 555-4567"},{"type":"mobile","number":"123 456-7890"}],"children":[],"spouse":null}`)

// CompressorGzip
func TestCompressorGzip(t *testing.T) {
	res := Response{body: zipTest}
	c := CompressorGzip{}
	crRes := c.Compress(res)
	if len(res.body) <= len(crRes.body) {
		t.Fatal("No Compression in Gzip")
	}
	exRes := c.Expand(crRes)
	if !bytes.Equal(res.body, exRes.body) {
		t.Fatal("Expanded compression does not match in Gzip")
	}
}

// CompressorSnappy
func TestCompressorSnappy(t *testing.T) {
	res := Response{body: zipTest}
	c := CompressorSnappy{}
	crRes := c.Compress(res)
	if len(res.body) <= len(crRes.body) {
		t.Fatal("No Compression in Gzip")
	}
	exRes := c.Expand(crRes)
	if !bytes.Equal(res.body, exRes.body) {
		t.Fatal("Expanded compression does not match in Gzip")
	}
}
