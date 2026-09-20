// Package hash provides HMAC-SHA256 request/response signing used to detect
// tampering between the agent and the server.
package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func sum(data []byte, key string) []byte {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return h.Sum(nil)
}

// Sign returns the hex-encoded HMAC-SHA256 signature of data under key.
func Sign(data []byte, key string) string {
	return hex.EncodeToString(sum(data, key))
}

// Valid reports whether got is the correct hex-encoded HMAC-SHA256
// signature of data under key. It returns false, without panicking, if got
// is not valid hex.
func Valid(data []byte, key, got string) bool {
	decoded, err := hex.DecodeString(got)
	if err != nil {
		return false
	}

	return hmac.Equal(sum(data, key), decoded)
}
