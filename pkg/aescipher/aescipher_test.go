package aescipher

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"testing"
)

// randomKey returns a fresh 32‑byte AES‑256 key generated from crypto/rand.
func randomKey(t *testing.T) []byte {
	t.Helper()
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return k
}

// randomData returns n random bytes using crypto/rand.
func randomData(t *testing.T, n int) []byte {
	t.Helper()
	d := make([]byte, n)
	if _, err := rand.Read(d); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return d
}

// TestRoundTrip encrypts and then decrypts a plaintext and expects to get the
// original bytes back unchanged.
func TestRoundTrip(t *testing.T) {
	key := hex.EncodeToString(randomKey(t))
	enc, err := New(key)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	plain := []byte("Hello, world!")

	ct, err := enc.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	got, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("decrypt mismatch: want %q, got %q", plain, got)
	}
}

// TestWrongKeySize ensures New returns an error when the key length is not 32 bytes.
func TestWrongKeySize(t *testing.T) {
	key := "too‑short"
	if _, err := New(key); err == nil {
		t.Fatal("expected error for invalid key length, got nil")
	}
}

// TestTamper flips a byte in the ciphertext and expects an authentication error on Decrypt.
func TestTamper(t *testing.T) {
	key := hex.EncodeToString(randomKey(t))
	enc, _ := New(key)
	pt := []byte("secret")
	ct, _ := enc.Encrypt(pt)
	ct[len(ct)-1] ^= 0xFF // corrupt the last byte
	if _, err := enc.Decrypt(ct); err == nil {
		t.Fatal("expected authentication error, got nil")
	}
}

// BenchmarkEncrypt times Encrypt on a 1 KiB buffer.
func BenchmarkEncrypt(b *testing.B) {
	key := hex.EncodeToString(randomKey(nil)) // zero key avoids allocating in the loop
	enc, _ := New(key)
	data := randomData(nil, 1<<10) // 1 KiB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enc.Encrypt(data)
	}
}

// BenchmarkDecrypt times Decrypt on a 1 KiB buffer.
func BenchmarkDecrypt(b *testing.B) {
	key := hex.EncodeToString(randomKey(nil))
	enc, _ := New(key)
	data := randomData(nil, 1<<10)
	ct, _ := enc.Encrypt(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enc.Decrypt(ct)
	}
}
