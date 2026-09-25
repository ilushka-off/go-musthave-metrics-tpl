package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/hash"
	"go.uber.org/zap"
)

type hashResponseWriter struct {
	http.ResponseWriter
	body       bytes.Buffer
	statusCode int
}

func (w *hashResponseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *hashResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

// Hash returns middleware that verifies the "HashSHA256" request header (if
// present) against an HMAC of the body signed with key, rejecting mismatches
// with 400, and signs every response body with the same key in its own
// "HashSHA256" header. If key is empty, the middleware is a no-op.
func Hash(key string, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}
			gotHash := r.Header.Get("HashSHA256")
			if gotHash != "" {
				data, err := io.ReadAll(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(data))

				if !hash.Valid(data, key, gotHash) {
					w.WriteHeader(http.StatusBadRequest)
					log.Error("Status Bad Request")
					return
				}
			}

			hw := &hashResponseWriter{
				ResponseWriter: w,
			}
			next.ServeHTTP(hw, r)

			signature := hash.Sign(hw.body.Bytes(), key)
			w.Header().Set("HashSHA256", signature)
			statusCode := hw.statusCode
			if statusCode == 0 {
				statusCode = http.StatusOK
			}
			w.WriteHeader(statusCode)
			if _, err := w.Write(hw.body.Bytes()); err != nil {
				log.Error("failed to write response", zap.Error(err))
			}
		})
	}
}
