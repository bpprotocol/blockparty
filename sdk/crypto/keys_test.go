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

func TestMakeMLDSAPairDeterministic(t *testing.T) {
	a := MakeMLDSAPair(testSeed)
	b := MakeMLDSAPair(testSeed)
	if !bytes.Equal(a.PublicBytes(), b.PublicBytes()) {
		t.Fatal("MakeMLDSAPair public keys differ across runs for the same seed")
	}
	if !a.Private.Equal(b.Private) {
		t.Fatal("MakeMLDSAPair private keys differ across runs for the same seed")
	}
}

func TestDifferentSeedsDifferentKeys(t *testing.T) {
	if bytes.Equal(MakeKyberPair([]byte("a")).PublicBytes(), MakeKyberPair([]byte("b")).PublicBytes()) {
		t.Error("different seeds yielded identical Kyber public keys")
	}
	if bytes.Equal(MakeMLDSAPair([]byte("a")).PublicBytes(), MakeMLDSAPair([]byte("b")).PublicBytes()) {
		t.Error("different seeds yielded identical ML-DSA public keys")
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
	kp := MakeMLDSAPair(testSeed)
	msg := []byte("nothing can stop the signal")
	sig := kp.Sign(msg)
	if !VerifyMLDSA(kp.Public, msg, sig) {
		t.Fatal("valid signature failed to verify")
	}
	if VerifyMLDSA(kp.Public, []byte("tampered"), sig) {
		t.Fatal("signature verified against a different message")
	}
	bad := append([]byte(nil), sig...)
	bad[0] ^= 0xff
	if VerifyMLDSA(kp.Public, msg, bad) {
		t.Fatal("tampered signature verified")
	}
	other := MakeMLDSAPair([]byte("other"))
	if VerifyMLDSA(other.Public, msg, sig) {
		t.Fatal("signature verified under the wrong public key")
	}
}

// TestGoldenPublicKeys locks the derived public keys for a fixed seed so an
// accidental change to the derivation (library swap, seed handling) is caught.
// These are conformance vectors other clients can reproduce.
func TestGoldenPublicKeys(t *testing.T) {
	const (
		goldenKyber = GOLDEN_KYBER
		goldenMLDSA = GOLDEN_MLDSA
	)
	if got := hex.EncodeToString(MakeKyberPair(testSeed).PublicBytes()); got != goldenKyber {
		t.Errorf("Kyber public key changed:\n got  %s\n want %s", got, goldenKyber)
	}
	if got := hex.EncodeToString(MakeMLDSAPair(testSeed).PublicBytes()); got != goldenMLDSA {
		t.Errorf("ML-DSA public key changed:\n got  %s\n want %s", got, goldenMLDSA)
	}
}
