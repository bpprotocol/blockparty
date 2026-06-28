package derive

import (
	"encoding/hex"
	"testing"

	"github.com/bpprotocol/blockparty/sdk/crypto"
)

const (
	testPhrase   = "bpprotocol.org/v1/test-world"
	testType     = "bpprotocol.org/v1/types/content.post"
	testAudience = "bpprotocol.org/v1/audience/public-1"
)

func TestOpenWorldDeterministic(t *testing.T) {
	a := OpenWorld(testPhrase)
	b := OpenWorld(testPhrase)
	if hex.EncodeToString(a.WalletSalt) != hex.EncodeToString(b.WalletSalt) ||
		hex.EncodeToString(a.TypeSalt) != hex.EncodeToString(b.TypeSalt) ||
		hex.EncodeToString(a.AudienceSalt) != hex.EncodeToString(b.AudienceSalt) {
		t.Fatal("salts differ across runs for the same phrase")
	}
	if hex.EncodeToString(a.SigningKey.PublicBytes()) != hex.EncodeToString(b.SigningKey.PublicBytes()) {
		t.Fatal("world signing key differs across runs for the same phrase")
	}
}

func TestDifferentWorldSeeds(t *testing.T) {
	a := OpenWorld("world-a")
	b := OpenWorld("world-b")
	if hex.EncodeToString(a.TypeSalt) == hex.EncodeToString(b.TypeSalt) {
		t.Error("different phrases produced the same TypeSalt")
	}
	if hex.EncodeToString(a.SigningKey.PublicBytes()) == hex.EncodeToString(b.SigningKey.PublicBytes()) {
		t.Error("different phrases produced the same signing key")
	}
}

func TestCodesAreWorldScoped(t *testing.T) {
	a := OpenWorld("world-a")
	b := OpenWorld("world-b")
	if GetTypeCode(a, testType).Hex() == GetTypeCode(b, testType).Hex() {
		t.Error("same type URN hashed identically across different Worlds")
	}
	if GetAudienceCode(a, testAudience).Hex() == GetAudienceCode(b, testAudience).Hex() {
		t.Error("same audience ID hashed identically across different Worlds")
	}
	// Stable within a World.
	if GetTypeCode(a, testType).Hex() != GetTypeCode(a, testType).Hex() {
		t.Error("type code not stable within a World")
	}
	if len(GetTypeCode(a, testType)) != CodeBytes {
		t.Errorf("code length = %d, want %d", len(GetTypeCode(a, testType)), CodeBytes)
	}
}

func TestCodeNormalization(t *testing.T) {
	w := OpenWorld(testPhrase)
	base := GetTypeCode(w, testType).Hex()
	if GetTypeCode(w, "  "+testType+"  ").Hex() != base {
		t.Error("surrounding whitespace changed the type code")
	}
	if GetTypeCode(w, testType+"/").Hex() != base {
		t.Error("trailing slash changed the type code")
	}
}

func TestBytesToAddress(t *testing.T) {
	dil := []byte("ml-dsa-public-key-bytes")
	kyb := []byte("kyber-public-key-bytes")
	addr := BytesToAddress(dil, kyb)
	if len(addr) != 40 {
		t.Fatalf("address length = %d, want 40 hex chars", len(addr))
	}
	// Matches a manual computation of the documented formula.
	digest := crypto.Keccak256([]byte("v1"), dil, kyb)
	want := hex.EncodeToString(digest[len(digest)-20:])
	if string(addr) != want {
		t.Fatalf("address = %s, want %s", addr, want)
	}
	// Sensitive to each key.
	if BytesToAddress(dil, kyb) == BytesToAddress(append([]byte{0}, dil...), kyb) {
		t.Error("address not sensitive to ml-dsa key")
	}
	if BytesToAddress(dil, kyb) == BytesToAddress(dil, append([]byte{0}, kyb...)) {
		t.Error("address not sensitive to kyber key")
	}
}

func TestBlockIDContentBinding(t *testing.T) {
	w := OpenWorld(testPhrase)
	tc := GetTypeCode(w, testType)
	ac := GetAudienceCode(w, testAudience)
	addr := BytesToAddress([]byte("d"), []byte("k"))

	id1 := GetBlockID(1, 1700000000, ac, addr, tc, []byte("hello"))
	id1b := GetBlockID(1, 1700000000, ac, addr, tc, []byte("hello"))
	id2 := GetBlockID(1, 1700000000, ac, addr, tc, []byte("hello!")) // payload changed

	if id1 != id1b {
		t.Fatal("block ID not deterministic for identical inputs")
	}
	if id1 == id2 {
		t.Fatal("block ID did not change when the payload changed (not content-binding)")
	}
	if len(id1) != 64 { // SHA-256 hex
		t.Fatalf("block ID length = %d, want 64", len(id1))
	}
}
