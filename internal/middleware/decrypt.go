package middleware

import (
	"bytes"
	"crypto/rsa"
	"errors"
	"io"
	"net/http"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/crypto"
	"go.uber.org/zap"
)

// maxEncryptedBodySize limits how much of an encrypted request body is read
// into memory. A metrics batch from the agent is a few kilobytes, so 1 MiB
// leaves plenty of headroom.
const maxEncryptedBodySize = 1 << 20

// Decrypt returns middleware that decrypts request bodies encrypted by the
// agent with the matching public key, rejecting undecryptable bodies with
// 400 and bodies larger than maxEncryptedBodySize with 413. If priv is nil,
// the middleware is a no-op.
//
// It is mounted on all routes, including /debug/pprof, on purpose: with a
// key configured every request body is expected to be encrypted. The pprof
// endpoints are plain GETs without a body, so they pass through untouched.
func Decrypt(priv *rsa.PrivateKey, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if priv == nil || r.Body == nil || r.Body == http.NoBody {
				next.ServeHTTP(w, r)
				return
			}

			data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxEncryptedBodySize))
			if err != nil {
				var tooLarge *http.MaxBytesError
				if errors.As(err, &tooLarge) {
					http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
					return
				}
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}

			plain, err := crypto.Decrypt(priv, data)
			if err != nil {
				log.Error("failed to decrypt request body", zap.Error(err))
				http.Error(w, "failed to decrypt request body", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(plain))
			r.ContentLength = int64(len(plain))
			next.ServeHTTP(w, r)
		})
	}
}
