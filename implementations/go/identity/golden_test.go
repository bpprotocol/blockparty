package identity

import (
	"encoding/hex"
	"testing"

	"github.com/bpprotocol/blockparty/implementations/go/crypto"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
)

// Golden conformance vectors for OpenIdentity(OpenWorld(testPhrase),
// testPassphrase). Keys are fingerprinted via Keccak256.
const (
	goldenAddress = "51fdf6cd0baf5eb11bd1d1aa367e838eb6f8f685"
	goldenMLDSAFP = "76542c4997870c0acecd929967ac3cc2b66803ff23bdf6394072c0cfc6724407"
	goldenKyberFP = "48d8f10cb682fe5009d73cd7686a7ce8c8c914d153f803f90b0ee5291ed7f38c"
)

func TestGoldenIdentity(t *testing.T) {
	id := OpenIdentity(derive.OpenWorld(testPhrase), testPassphrase)
	if string(id.Address) != goldenAddress {
		t.Errorf("Address changed:\n got  %s\n want %s", id.Address, goldenAddress)
	}
	if got := hex.EncodeToString(crypto.Keccak256(id.MLDSA.PublicBytes())); got != goldenMLDSAFP {
		t.Errorf("ML-DSA key changed:\n got  %s\n want %s", got, goldenMLDSAFP)
	}
	if got := hex.EncodeToString(crypto.Keccak256(id.Kyber.PublicBytes())); got != goldenKyberFP {
		t.Errorf("Kyber key changed:\n got  %s\n want %s", got, goldenKyberFP)
	}
}
