package block

import (
	"encoding/hex"
	"testing"

	"github.com/bpprotocol/blockparty/sdk/crypto"
)

// Golden conformance vectors for a fully-signed block over fixed inputs. Both
// the world signing key and Dilithium signatures are deterministic, so the
// encoded block is reproducible; we lock its ID and a fingerprint of its
// canonical encoding (sans co-signatures).
const (
	goldenBlockIDHex = "80697805105d6d25a399327203d28df0bc6364d9651e53cea9e404f7f88d4e02"
	goldenBlockFP    = "f74d7e7ef4a9e1304a42fb6a9ea746d2bfc6edd3e8b5d95a7c1771310d74ab09"
)

func TestGoldenSignedBlock(t *testing.T) {
	f := newFixture()
	b := f.signed([]byte("hello"))

	if got := IDHex(b); got != goldenBlockIDHex {
		t.Errorf("block ID changed:\n got  %s\n want %s", got, goldenBlockIDHex)
	}
	enc, err := Encode(b)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if got := hex.EncodeToString(crypto.Keccak256(enc)); got != goldenBlockFP {
		t.Errorf("signed-block encoding changed:\n got  %s\n want %s", got, goldenBlockFP)
	}
}
