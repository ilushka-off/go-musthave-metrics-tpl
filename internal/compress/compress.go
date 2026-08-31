package compress

import (
	"bytes"
	"compress/gzip"
	"io"
)

func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		gz.Close()
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func NewReader(r io.Reader) (io.ReadCloser, error) {
	return gzip.NewReader(r)
}
