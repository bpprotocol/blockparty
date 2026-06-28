package derive

import (
	"encoding/hex"
	"testing"

	"github.com/bpprotocol/blockparty/sdk/crypto"
)

// Golden conformance vectors for testPhrase / testType / testAudience, locked
// so any change to the derivations is caught and other clients can reproduce
// them. The world signing key (1952 bytes) is fingerprinted via Keccak256.
const (
	goldenWalletSalt   = "1ddf42bd8f7a2595044f9f54b06bc74f432aa4ea0a88fe205b6031f0499d4623"
	goldenTypeSalt     = "242b4984c7100a1e3ce37745955ba1ceeeb32e16c4c844ffee3ef83b8e91aa4c"
	goldenAudienceSalt = "d3c3a17a70b1039d633b7f3b05c64ae06aa88482207314342ded010196d9b7ff"
	goldenSigningKeyFP = "c107da6ad16905ec7ededb15d88f93cf81c2f51f18df7e53bf8c2e49fc698fb6"
	goldenTypeCode     = "a451d8b7e0ec778506b8001ebd31c2b9"
	goldenAudienceCode = "8572d37a189d92544082f34254f01d8e"
	goldenBlockID      = "290201fe18e78a4ca31574acb3530cb838816de8c76f5669adb5f9111182aa13"
)

func TestGoldenDerivations(t *testing.T) {
	w := OpenWorld(testPhrase)
	check := func(name, got, want string) {
		if got != want {
			t.Errorf("%s changed:\n got  %s\n want %s", name, got, want)
		}
	}
	check("WalletSalt", hex.EncodeToString(w.WalletSalt), goldenWalletSalt)
	check("TypeSalt", hex.EncodeToString(w.TypeSalt), goldenTypeSalt)
	check("AudienceSalt", hex.EncodeToString(w.AudienceSalt), goldenAudienceSalt)
	check("SigningKeyFP", hex.EncodeToString(crypto.Keccak256(w.SigningKey.PublicBytes())), goldenSigningKeyFP)
	check("TypeCode", GetTypeCode(w, testType).Hex(), goldenTypeCode)
	check("AudienceCode", GetAudienceCode(w, testAudience).Hex(), goldenAudienceCode)

	addr := BytesToAddress([]byte("d"), []byte("k"))
	id := GetBlockID(1, 1700000000, GetAudienceCode(w, testAudience), addr, GetTypeCode(w, testType), []byte("hello"))
	check("BlockID", id, goldenBlockID)
}
