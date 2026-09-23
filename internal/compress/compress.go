// Package compress provides thin gzip helpers used by the HTTP compression
// middleware.
package compress

import (
	"bytes"
	"compress/gzip"
	"io"
)

// Compress gzip-compresses data and returns the compressed bytes.
func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		_ = gz.Close()
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// NewReader wraps r in a gzip.Reader that decompresses on Read. The caller
// is responsible for closing the returned reader.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	return gzip.NewReader(r)
}
