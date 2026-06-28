package identity

import (
	"testing"

	"github.com/bpprotocol/blockparty/sdk/crypto"
	"github.com/bpprotocol/blockparty/sdk/derive"
)

const (
	testPhrase     = "bpprotocol.org/v1/test-world"
	testPassphrase = "correct horse battery staple"
)

func TestOpenIdentityDeterministic(t *testing.T) {
	w := derive.OpenWorld(testPhrase)
	a := OpenIdentity(w, testPassphrase)
	b := OpenIdentity(w, testPassphrase)
	if a.Address != b.Address {
		t.Fatal("address differs across runs for the same world + passphrase")
	}
	if string(a.MLDSA.PublicBytes()) != string(b.MLDSA.PublicBytes()) {
		t.Fatal("ML-DSA key differs across runs")
	}
	if string(a.Kyber.PublicBytes()) != string(b.Kyber.PublicBytes()) {
		t.Fatal("kyber key differs across runs")
	}
	if len(a.Address) != 40 {
		t.Fatalf("address length = %d, want 40", len(a.Address))
	}
}

func TestOpenIdentityWorldScoped(t *testing.T) {
	a := OpenIdentity(derive.OpenWorld("world-a"), testPassphrase)
	b := OpenIdentity(derive.OpenWorld("world-b"), testPassphrase)
	if a.Address == b.Address {
		t.Fatal("same passphrase produced the same identity across different Worlds")
	}
}

func TestOpenIdentityPassphraseSensitive(t *testing.T) {
	w := derive.OpenWorld(testPhrase)
	if OpenIdentity(w, "alice").Address == OpenIdentity(w, "bob").Address {
		t.Fatal("different passphrases produced the same identity")
	}
}

func TestDerivedKeysAreUsable(t *testing.T) {
	w := derive.OpenWorld(testPhrase)
	id := OpenIdentity(w, testPassphrase)

	// ML-DSA key signs and verifies.
	msg := []byte("nothing can stop the signal")
	sig := id.MLDSA.Sign(msg)
	if !crypto.VerifyMLDSA(id.MLDSA.Public, msg, sig) {
		t.Fatal("derived ML-DSA key failed sign/verify")
	}

	// Kyber key encapsulates and decapsulates.
	ct, ss, err := crypto.Encapsulate(id.Kyber.Public)
	if err != nil {
		t.Fatalf("encapsulate: %v", err)
	}
	ss2, err := id.Kyber.Decapsulate(ct)
	if err != nil {
		t.Fatalf("decapsulate: %v", err)
	}
	if string(ss) != string(ss2) {
		t.Fatal("derived kyber key failed encaps/decaps round-trip")
	}

	// Address is exactly BytesToAddress of the two public keys.
	want := derive.BytesToAddress(id.MLDSA.PublicBytes(), id.Kyber.PublicBytes())
	if id.Address != want {
		t.Fatalf("address = %s, want %s", id.Address, want)
	}
}
