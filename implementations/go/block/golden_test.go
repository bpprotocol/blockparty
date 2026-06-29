package block

import (
	"encoding/hex"
	"testing"

	"github.com/bpprotocol/blockparty/implementations/go/crypto"
)

// Golden conformance vectors for a fully-signed block over fixed inputs. Both
// the world signing key and ML-DSA signatures are deterministic, so the
// encoded block is reproducible; we lock its ID and a fingerprint of its
// canonical encoding (sans co-signatures).
const (
	goldenBlockIDHex = "ebfe295bf1c8ef8f016281eb463ba296f807dbcf8fd5f34108c8f6dbcef839a8"
	goldenBlockFP    = "b8284a391dbe75a525cb41b6894f04b1b7d378b11d440de2d9f0f7e71529f9dc"
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
