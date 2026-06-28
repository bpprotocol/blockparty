package identity

import (
	"encoding/hex"
	"testing"

	"github.com/bpprotocol/blockparty/sdk/crypto"
	"github.com/bpprotocol/blockparty/sdk/derive"
)

// Golden conformance vectors for OpenIdentity(OpenWorld(testPhrase),
// testPassphrase). Keys are fingerprinted via Keccak256.
const (
	goldenAddress     = "cff62cff35a0cc4271c262c07a86cbccc63ad13a"
	goldenDilithiumFP = "d24f70f809d782cec7b3bcc6387fb0f97f80a9b57984c99df3192023fa77a716"
	goldenKyberFP     = "48d8f10cb682fe5009d73cd7686a7ce8c8c914d153f803f90b0ee5291ed7f38c"
)

func TestGoldenIdentity(t *testing.T) {
	id := OpenIdentity(derive.OpenWorld(testPhrase), testPassphrase)
	if string(id.Address) != goldenAddress {
		t.Errorf("Address changed:\n got  %s\n want %s", id.Address, goldenAddress)
	}
	if got := hex.EncodeToString(crypto.Keccak256(id.Dilithium.PublicBytes())); got != goldenDilithiumFP {
		t.Errorf("Dilithium key changed:\n got  %s\n want %s", got, goldenDilithiumFP)
	}
	if got := hex.EncodeToString(crypto.Keccak256(id.Kyber.PublicBytes())); got != goldenKyberFP {
		t.Errorf("Kyber key changed:\n got  %s\n want %s", got, goldenKyberFP)
	}
}
