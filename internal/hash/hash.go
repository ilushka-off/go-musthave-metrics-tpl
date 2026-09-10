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

func Sign(data []byte, key string) string {
	return hex.EncodeToString(sum(data, key))
}

func Valid(data []byte, key, got string) bool {
	decoded, err := hex.DecodeString(got)
	if err != nil {
		return false
	}

	return hmac.Equal(sum(data, key), decoded)
}
