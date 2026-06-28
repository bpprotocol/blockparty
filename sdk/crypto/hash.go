package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/sha3"
)

// Keccak256 returns the 32-byte Keccak-256 digest of the concatenation of the
// given byte slices. This is the original Keccak padding (Ethereum-style), not
// FIPS-202 SHA3-256.
func Keccak256(parts ...[]byte) []byte {
	h := sha3.NewLegacyKeccak256()
	for _, p := range parts {
		h.Write(p)
	}
	return h.Sum(nil)
}

// Sha256 returns the 32-byte SHA-256 digest of the concatenation of the given
// byte slices.
func Sha256(parts ...[]byte) []byte {
	h := sha256.New()
	for _, p := range parts {
		h.Write(p)
	}
	return h.Sum(nil)
}

// HMACSHA256 returns the HMAC-SHA256 of message under key.
func HMACSHA256(key, message []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(message)
	return m.Sum(nil)
}

// HKDFSHA256 derives length bytes of key material from the input keying
// material using HKDF (extract-and-expand) with SHA-256. salt and info may be
// nil. It panics only if length exceeds HKDF's 255*HashLen ceiling, which is a
// programming error for the fixed-size derivations used by the protocol.
func HKDFSHA256(ikm, salt, info []byte, length int) []byte {
	r := hkdf.New(sha256.New, ikm, salt, info)
	out := make([]byte, length)
	if _, err := io.ReadFull(r, out); err != nil {
		panic(fmt.Sprintf("crypto: HKDFSHA256 length %d too large: %v", length, err))
	}
	return out
}
