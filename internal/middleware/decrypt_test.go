package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/crypto"
	"go.uber.org/zap"
)

func TestDecrypt(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	enc, err := crypto.Encrypt(&priv.PublicKey, []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}

	var got []byte
	h := Decrypt(priv, zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
	}))

	tests := []struct {
		name string
		body []byte
		want int
	}{
		{"valid", enc, http.StatusOK},
		{"garbage", []byte("garbage"), http.StatusBadRequest},
		{"too large", make([]byte, maxEncryptedBodySize+1), http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got = nil
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(tt.body)))
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
			if tt.want == http.StatusOK && string(got) != "hello" {
				t.Fatalf("body = %q", got)
			}
		})
	}
}
