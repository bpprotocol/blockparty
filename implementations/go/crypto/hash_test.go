package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

func TestKeccak256KnownAnswers(t *testing.T) {
	// Keccak-256 (Ethereum-style padding), distinct from FIPS SHA3-256.
	cases := []struct{ in, want string }{
		{"", "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"},
		{"abc", "4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45"},
	}
	for _, c := range cases {
		got := hex.EncodeToString(Keccak256([]byte(c.in)))
		if got != c.want {
			t.Errorf("Keccak256(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestSha256KnownAnswer(t *testing.T) {
	got := hex.EncodeToString(Sha256([]byte("")))
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("Sha256(\"\") = %s, want %s", got, want)
	}
}

func TestKeccak256MultiPartConcatenation(t *testing.T) {
	if !bytes.Equal(Keccak256([]byte("foo"), []byte("bar")), Keccak256([]byte("foobar"))) {
		t.Error("Keccak256 multi-part should equal hashing the concatenation")
	}
}

func TestHMACSHA256RFC4231Case1(t *testing.T) {
	key := bytes.Repeat([]byte{0x0b}, 20)
	got := hex.EncodeToString(HMACSHA256(key, []byte("Hi There")))
	want := "b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7"
	if got != want {
		t.Errorf("HMAC-SHA256 = %s, want %s", got, want)
	}
}

func TestHKDFSHA256RFC5869Case1(t *testing.T) {
	ikm := bytes.Repeat([]byte{0x0b}, 22)
	salt := mustHex(t, "000102030405060708090a0b0c")
	info := mustHex(t, "f0f1f2f3f4f5f6f7f8f9")
	got := hex.EncodeToString(HKDFSHA256(ikm, salt, info, 42))
	want := "3cb25f25faacd57a90434f64d0362f2a2d2d0a90cf1a5a4c5db02d56ecc4c5bf34007208d5b887185865"
	if got != want {
		t.Errorf("HKDF-SHA256 = %s, want %s", got, want)
	}
}
