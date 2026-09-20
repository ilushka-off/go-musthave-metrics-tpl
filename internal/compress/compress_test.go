package compress

import (
	"bytes"
	"io"
	"testing"
)

func TestCompress_RoundTrip(t *testing.T) {
	original := []byte("the quick brown fox jumps over the lazy dog, repeated: " +
		"the quick brown fox jumps over the lazy dog")

	compressed, err := Compress(original)
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	r, err := NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read decompressed data: %v", err)
	}

	if !bytes.Equal(decompressed, original) {
		t.Fatalf("decompressed = %q; want %q", decompressed, original)
	}
}

func TestCompress_EmptyInput(t *testing.T) {
	compressed, err := Compress([]byte{})
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	r, err := NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read decompressed data: %v", err)
	}

	if len(decompressed) != 0 {
		t.Fatalf("decompressed = %q; want empty", decompressed)
	}
}

func TestNewReader_InvalidData(t *testing.T) {
	_, err := NewReader(bytes.NewReader([]byte("not gzip data")))
	if err == nil {
		t.Fatal("NewReader() error = nil; want non-nil for non-gzip input")
	}
}
