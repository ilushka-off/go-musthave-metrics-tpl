package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	for _, size := range []int{0, 1, 190, 191, 5000} {
		data := make([]byte, size)
		_, _ = rand.Read(data)

		enc, err := Encrypt(&priv.PublicKey, data)
		if err != nil {
			t.Fatalf("size %d: encrypt: %v", size, err)
		}
		dec, err := Decrypt(priv, enc)
		if err != nil {
			t.Fatalf("size %d: decrypt: %v", size, err)
		}
		if !bytes.Equal(dec, data) {
			t.Fatalf("size %d: round trip mismatch", size)
		}
	}
}

func TestDecryptRejectsGarbage(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(priv, []byte("not encrypted")); err == nil {
		t.Fatal("expected error")
	}
	if _, err := Decrypt(priv, make([]byte, 256)); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadKeys(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()

	privPath := filepath.Join(dir, "private.pem")
	pubPath := filepath.Join(dir, "public.pem")
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	write := func(path, typ string, der []byte) {
		if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(privPath, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(priv))
	write(pubPath, "PUBLIC KEY", pubDER)

	gotPriv, err := LoadPrivateKey(privPath)
	if err != nil || !gotPriv.Equal(priv) {
		t.Fatalf("load private: %v", err)
	}
	gotPub, err := LoadPublicKey(pubPath)
	if err != nil || !gotPub.Equal(&priv.PublicKey) {
		t.Fatalf("load public: %v", err)
	}
	if _, err := LoadPublicKey(filepath.Join(dir, "missing.pem")); err == nil {
		t.Fatal("expected error for missing file")
	}
}
