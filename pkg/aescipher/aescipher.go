// Package aescipher provides a minimal, opinionated wrapper around AES-GCM
// to make authenticated encryption and decryption straightforward.
package aescipher

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

const (
	// NonceSizeGCM is the recommended size for AES-GCM nonces.
	NonceSizeGCM  = 12
	versionPrefix = "v1" // 2-byte marker identifying ciphertexts of this library
)

// Cryptor defines the minimal interface for an authenticated symmetric cipher.
type Cryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)

	IsEncryptedString(b64 string) bool
	EncryptString(plain string) (string, error)
	DecryptString(b64 string) (string, error)
}

type gcmCryptor struct {
	aead cipher.AEAD
}

// New returns a Cryptor backed by AES-GCM.
// Key must be 16, 24, or 32 bytes long (hex-encoded).
func New(key string) (Cryptor, error) {
	keyDecoded, err := hex.DecodeString(key)
	if err != nil {
		return nil, err
	}
	keyLen := len(keyDecoded)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return nil, errors.New("aescipher: key length must be 16, 24, or 32 bytes")
	}

	block, err := aes.NewCipher(keyDecoded)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &gcmCryptor{aead: aead}, nil
}

// EncryptString encrypts a UTF-8 string and returns Base64.
func (g *gcmCryptor) EncryptString(plain string) (string, error) {
	ct, err := g.Encrypt([]byte(plain))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ct), nil
}

// DecryptString decrypts Base64 and returns UTF-8.
func (g *gcmCryptor) DecryptString(b64 string) (string, error) {
	ct, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	pt, err := g.Decrypt(ct)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// Encrypt encrypts plaintext.
// Layout : "v1" | nonce (12) | ciphertext+tag (Seal output).
func (g *gcmCryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, NonceSizeGCM)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal with AAD = versionPrefix.
	enc := g.aead.Seal(nil, nonce, plaintext, []byte(versionPrefix))

	out := make([]byte, 0, len(versionPrefix)+len(nonce)+len(enc))
	out = append(out, versionPrefix...)
	out = append(out, nonce...)
	out = append(out, enc...)
	return out, nil
}

// Decrypt decrypts data created by Encrypt.
func (g *gcmCryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	minLen := len(versionPrefix) + NonceSizeGCM + g.aead.Overhead()
	if len(ciphertext) < minLen {
		return nil, errors.New("aescipher: ciphertext too short")
	}
	if !bytes.HasPrefix(ciphertext, []byte(versionPrefix)) {
		return nil, errors.New("aescipher: invalid prefix")
	}

	offset := len(versionPrefix)
	nonce := ciphertext[offset : offset+NonceSizeGCM]
	enc := ciphertext[offset+NonceSizeGCM:]

	plaintext, err := g.aead.Open(nil, nonce, enc, []byte(versionPrefix))
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// IsEncryptedString returns true if the string is an encrypted string
// produced by Encrypt (false positive probability ≈ 2⁻¹²⁸).
func (g *gcmCryptor) IsEncryptedString(b64 string) bool {
	ct, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return false
	}

	minLen := len(versionPrefix) + NonceSizeGCM + g.aead.Overhead()
	if len(ct) < minLen || !bytes.HasPrefix(ct, []byte(versionPrefix)) {
		return false
	}

	offset := len(versionPrefix)
	nonce := ct[offset : offset+NonceSizeGCM]
	enc := ct[offset+NonceSizeGCM:]

	if _, err = g.aead.Open(nil, nonce, enc, []byte(versionPrefix)); err != nil {
		return false
	}
	return true
}
