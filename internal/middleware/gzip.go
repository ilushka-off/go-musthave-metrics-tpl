package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func GzipDecompress() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-Encoding") == "gzip" {
				gzReader, err := gzip.NewReader(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				defer gzReader.Close()
				r.Body = gzReader
			}

			next.ServeHTTP(w, r)
			return
		})
	}
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	contentType := w.Header().Get("Content-Type")

	if contentType == "application/json" || contentType == "text/html" {
		w.gz = gzip.NewWriter(w.ResponseWriter)
		w.Header().Set("Content-Encoding", "gzip")
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if w.gz != nil {
		return w.gz.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func GzipCompress() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				gzw := &gzipResponseWriter{ResponseWriter: w}
				next.ServeHTTP(gzw, r)
				if gzw.gz != nil {
					gzw.gz.Close()
				}
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}
