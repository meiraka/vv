package gzip

import (
	"bytes"
	"compress/gzip"
)

// Encode encodes bytes to gzipped data.
func Encode(data []byte) ([]byte, error) {
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, err := zw.Write(data)
	if err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return gz.Bytes(), nil
}
