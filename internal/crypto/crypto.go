// Package crypto implements asymmetric (RSA-OAEP) encryption of request
// bodies sent from the agent to the server.
//
// RSA can only encrypt messages shorter than the key size, so payloads are
// split into blocks which are encrypted independently and concatenated.
package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// LoadPublicKey reads a PEM-encoded RSA public key (PKIX or PKCS#1) from path.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	block, err := readPEM(path)
	if err != nil {
		return nil, err
	}

	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		key, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not an RSA key")
		}
		return key, nil
	}

	key, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return key, nil
}

// LoadPrivateKey reads a PEM-encoded RSA private key (PKCS#1 or PKCS#8) from path.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	block, err := readPEM(path)
	if err != nil {
		return nil, err
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not an RSA key")
	}
	return key, nil
}

func readPEM(path string) (*pem.Block, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("no PEM data found in key file")
	}
	return block, nil
}

// Encrypt encrypts data with pub using RSA-OAEP (SHA-256), block by block.
func Encrypt(pub *rsa.PublicKey, data []byte) ([]byte, error) {
	hash := sha256.New()
	chunk := pub.Size() - 2*hash.Size() - 2
	if chunk <= 0 {
		return nil, errors.New("rsa key is too small")
	}

	out := make([]byte, 0, (len(data)/chunk+1)*pub.Size())
	for start := 0; start < len(data); start += chunk {
		end := min(start+chunk, len(data))
		enc, err := rsa.EncryptOAEP(hash, rand.Reader, pub, data[start:end], nil)
		if err != nil {
			return nil, fmt.Errorf("encrypt block: %w", err)
		}
		out = append(out, enc...)
	}
	return out, nil
}

// Decrypt reverses Encrypt using the matching private key.
func Decrypt(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	size := priv.PublicKey.Size()
	if len(data)%size != 0 {
		return nil, errors.New("ciphertext length is not a multiple of key size")
	}

	hash := sha256.New()
	out := make([]byte, 0, len(data))
	for start := 0; start < len(data); start += size {
		dec, err := rsa.DecryptOAEP(hash, nil, priv, data[start:start+size], nil)
		if err != nil {
			return nil, fmt.Errorf("decrypt block: %w", err)
		}
		out = append(out, dec...)
	}
	return out, nil
}
