package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

var testSeed = []byte("bpprotocol.org/v1/test-identity")

func TestMakeKyberPairDeterministic(t *testing.T) {
	a := MakeKyberPair(testSeed)
	b := MakeKyberPair(testSeed)
	if !bytes.Equal(a.PublicBytes(), b.PublicBytes()) {
		t.Fatal("MakeKyberPair public keys differ across runs for the same seed")
	}
	if !a.Private.Equal(b.Private) {
		t.Fatal("MakeKyberPair private keys differ across runs for the same seed")
	}
}

func TestMakeDilithiumPairDeterministic(t *testing.T) {
	a := MakeDilithiumPair(testSeed)
	b := MakeDilithiumPair(testSeed)
	if !bytes.Equal(a.PublicBytes(), b.PublicBytes()) {
		t.Fatal("MakeDilithiumPair public keys differ across runs for the same seed")
	}
	if !a.Private.Equal(b.Private) {
		t.Fatal("MakeDilithiumPair private keys differ across runs for the same seed")
	}
}

func TestDifferentSeedsDifferentKeys(t *testing.T) {
	if bytes.Equal(MakeKyberPair([]byte("a")).PublicBytes(), MakeKyberPair([]byte("b")).PublicBytes()) {
		t.Error("different seeds yielded identical Kyber public keys")
	}
	if bytes.Equal(MakeDilithiumPair([]byte("a")).PublicBytes(), MakeDilithiumPair([]byte("b")).PublicBytes()) {
		t.Error("different seeds yielded identical Dilithium public keys")
	}
}

func TestEncapsulateDecapsulate(t *testing.T) {
	kp := MakeKyberPair(testSeed)
	ct, ss, err := Encapsulate(kp.Public)
	if err != nil {
		t.Fatalf("Encapsulate: %v", err)
	}
	ss2, err := kp.Decapsulate(ct)
	if err != nil {
		t.Fatalf("Decapsulate: %v", err)
	}
	if !bytes.Equal(ss, ss2) {
		t.Fatal("decapsulated shared secret does not match encapsulated one")
	}
	if len(ss) != kemScheme.SharedKeySize() {
		t.Fatalf("shared secret size = %d, want %d", len(ss), kemScheme.SharedKeySize())
	}
}

func TestDecapsulateWrongKeyDiffers(t *testing.T) {
	a := MakeKyberPair([]byte("a"))
	b := MakeKyberPair([]byte("b"))
	ct, ss, err := Encapsulate(a.Public)
	if err != nil {
		t.Fatalf("Encapsulate: %v", err)
	}
	// ML-KEM implicit rejection: decapsulating with the wrong key yields a
	// (deterministic) shared secret that does not match.
	ssWrong, err := b.Decapsulate(ct)
	if err != nil {
		t.Fatalf("Decapsulate: %v", err)
	}
	if bytes.Equal(ss, ssWrong) {
		t.Fatal("decapsulating with the wrong key matched the real shared secret")
	}
}

func TestSignVerify(t *testing.T) {
	kp := MakeDilithiumPair(testSeed)
	msg := []byte("nothing can stop the signal")
	sig := kp.Sign(msg)
	if !VerifyDilithium(kp.Public, msg, sig) {
		t.Fatal("valid signature failed to verify")
	}
	if VerifyDilithium(kp.Public, []byte("tampered"), sig) {
		t.Fatal("signature verified against a different message")
	}
	bad := append([]byte(nil), sig...)
	bad[0] ^= 0xff
	if VerifyDilithium(kp.Public, msg, bad) {
		t.Fatal("tampered signature verified")
	}
	other := MakeDilithiumPair([]byte("other"))
	if VerifyDilithium(other.Public, msg, sig) {
		t.Fatal("signature verified under the wrong public key")
	}
}

// TestGoldenPublicKeys locks the derived public keys for a fixed seed so an
// accidental change to the derivation (library swap, seed handling) is caught.
// These are conformance vectors other clients can reproduce.
func TestGoldenPublicKeys(t *testing.T) {
	const (
		goldenKyber     = GOLDEN_KYBER
		goldenDilithium = GOLDEN_DILITHIUM
	)
	if got := hex.EncodeToString(MakeKyberPair(testSeed).PublicBytes()); got != goldenKyber {
		t.Errorf("Kyber public key changed:\n got  %s\n want %s", got, goldenKyber)
	}
	if got := hex.EncodeToString(MakeDilithiumPair(testSeed).PublicBytes()); got != goldenDilithium {
		t.Errorf("Dilithium public key changed:\n got  %s\n want %s", got, goldenDilithium)
	}
}
