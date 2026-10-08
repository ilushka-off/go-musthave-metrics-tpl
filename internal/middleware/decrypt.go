package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/crypto"
	"go.uber.org/zap"
)

// Decrypt returns middleware that decrypts request bodies encrypted by the
// agent with the matching public key, rejecting undecryptable bodies with
// 400. If priv is nil, the middleware is a no-op.
func Decrypt(priv *rsa.PrivateKey, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if priv == nil || r.Body == nil || r.Body == http.NoBody {
				next.ServeHTTP(w, r)
				return
			}

			data, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			plain, err := crypto.Decrypt(priv, data)
			if err != nil {
				log.Error("failed to decrypt request body", zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(plain))
			r.ContentLength = int64(len(plain))
			next.ServeHTTP(w, r)
		})
	}
}
