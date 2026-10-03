package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/compress"
)

func TestGzipDecompress_ValidBodyIsDecompressed(t *testing.T) {
	compressed, err := compress.Compress([]byte("hello world"))
	if err != nil {
		t.Fatalf("compress.Compress() error = %v", err)
	}

	var gotBody []byte
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})

	wrapped := GzipDecompress()(inner)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(compressed))
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if string(gotBody) != "hello world" {
		t.Fatalf("body = %q, want %q", gotBody, "hello world")
	}
}

func TestGzipDecompress_InvalidBodyReturns400(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("inner handler must not run for a malformed gzip body")
	})

	wrapped := GzipDecompress()(inner)

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader("not gzip data"))
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGzipDecompress_NoHeaderPassesBodyThrough(t *testing.T) {
	var gotBody []byte
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})

	wrapped := GzipDecompress()(inner)

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader("plain body"))
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if string(gotBody) != "plain body" {
		t.Fatalf("body = %q, want %q", gotBody, "plain body")
	}
}

func TestGzipCompress_CompressesJSONWhenAccepted(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	wrapped := GzipCompress()(inner)

	req := httptest.NewRequest(http.MethodGet, "/value", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding = %q, want %q", rec.Header().Get("Content-Encoding"), "gzip")
	}

	gr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v", err)
	}
	defer func() { _ = gr.Close() }()

	data, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("failed to read decompressed body: %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Fatalf("decompressed body = %q, want %q", data, `{"ok":true}`)
	}
}

func TestGzipCompress_LeavesPlainTextUncompressed(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("plain"))
	})

	wrapped := GzipCompress()(inner)

	req := httptest.NewRequest(http.MethodGet, "/value", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("text/plain response must not be gzip-compressed")
	}
	if rec.Body.String() != "plain" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "plain")
	}
}

func TestGzipCompress_NoAcceptEncodingSkipsCompression(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	wrapped := GzipCompress()(inner)

	req := httptest.NewRequest(http.MethodGet, "/value", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("response must not be gzip-compressed without Accept-Encoding: gzip")
	}
	if rec.Body.String() != `{"ok":true}` {
		t.Fatalf("body = %q, want %q", rec.Body.String(), `{"ok":true}`)
	}
}
