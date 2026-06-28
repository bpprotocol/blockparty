package audiences

import (
	"bytes"
	"testing"

	"github.com/bpprotocol/blockparty/sdk/block"
	"github.com/bpprotocol/blockparty/sdk/derive"
	"github.com/bpprotocol/blockparty/sdk/encryption"
	"github.com/bpprotocol/blockparty/sdk/identity"
)

const (
	testWorldPhrase = "bpprotocol.org/v1/test-world"
	testType        = "bpprotocol.org/v1/types/content.post"
	testTimestamp   = int64(1700000000)
	// The reserved public-1 audience code in the test world, cross-checked
	// against the derive package's golden vector (#3).
	publicOneCodeHex = "8572d37a189d92544082f34254f01d8e"
)

func TestPublicAudienceConsistentWithDerive(t *testing.T) {
	w := derive.OpenWorld(testWorldPhrase)
	pub := PublicAudience(w, 1)
	if pub.ID != "bpprotocol.org/v1/audience/public-1" {
		t.Fatalf("unexpected ID %q", pub.ID)
	}
	if pub.Code.Hex() != derive.GetAudienceCode(w, pub.ID).Hex() {
		t.Fatal("PublicAudience code disagrees with derive.GetAudienceCode")
	}
	if pub.Code.Hex() != publicOneCodeHex {
		t.Fatalf("public-1 code = %s, want %s", pub.Code.Hex(), publicOneCodeHex)
	}
	if len(pub.Secret) != 32 {
		t.Fatalf("secret length = %d, want 32", len(pub.Secret))
	}
}

func TestPublicAudienceRoundTrip(t *testing.T) {
	w := derive.OpenWorld(testWorldPhrase)
	author := identity.OpenIdentity(w, "author")
	typeCode := derive.GetTypeCode(w, testType)
	pub := PublicAudience(w, 1)
	plaintext := []byte("town square message")

	data, err := encryption.EncryptForBlock(pub.Secret, block.Version, []byte(typeCode), []byte(pub.Code), testTimestamp, plaintext)
	if err != nil {
		t.Fatalf("EncryptForBlock: %v", err)
	}
	b := block.New(author.Address, typeCode, pub.Code, testTimestamp, data)
	block.Sign(b, w, author.Dilithium)

	// Another World member independently derives the audience via a registry
	// and decrypts.
	reg := NewRegistry()
	reg.Add(PublicAudience(w, 1))
	aud, ok := reg.ByCode(b.AudienceCode)
	if !ok {
		t.Fatal("registry did not resolve the block's audience code")
	}
	got, err := encryption.DecryptData(aud.Secret, b)
	if err != nil {
		t.Fatalf("member DecryptData: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("decrypted %q, want %q", got, plaintext)
	}

	// An outsider without the World seed derives a different secret and cannot
	// decrypt.
	outsider := derive.OpenWorld("a-different-world")
	wrong := PublicAudienceSecret(outsider, PublicAudienceID(1))
	if _, err := encryption.DecryptData(wrong, b); err != encryption.ErrDecrypt {
		t.Fatalf("outsider DecryptData = %v, want ErrDecrypt", err)
	}
}

func TestPublicAudienceWorldScoped(t *testing.T) {
	a := PublicAudience(derive.OpenWorld("world-a"), 1)
	b := PublicAudience(derive.OpenWorld("world-b"), 1)
	if a.Code.Hex() == b.Code.Hex() {
		t.Error("public-1 code identical across Worlds")
	}
	if bytes.Equal(a.Secret, b.Secret) {
		t.Error("public-1 secret identical across Worlds")
	}
}

func TestReservedSet(t *testing.T) {
	w := derive.OpenWorld(testWorldPhrase)
	set := ReservedPublicAudiences(w)
	if len(set) != ReservedPublicCount {
		t.Fatalf("reserved set size = %d, want %d", len(set), ReservedPublicCount)
	}
	seen := map[string]bool{}
	for _, a := range set {
		if seen[a.Code.Hex()] {
			t.Fatalf("duplicate code in reserved set: %s", a.Code.Hex())
		}
		seen[a.Code.Hex()] = true
	}
	if !IsReservedPublic(1) || !IsReservedPublic(16) || IsReservedPublic(0) || IsReservedPublic(17) {
		t.Error("IsReservedPublic boundaries wrong")
	}
}

func TestInboxAudience(t *testing.T) {
	w := derive.OpenWorld(testWorldPhrase)
	addr := derive.Address("cff62cff35a0cc4271c262c07a86cbccc63ad13a")

	in := InboxAudience(w, addr)
	if in.ID != "bpprotocol.org/v1/audience/inbox/"+string(addr) {
		t.Fatalf("unexpected inbox ID %q", in.ID)
	}
	if in.Secret != nil {
		t.Error("inbox audience should have no secret (rendezvous only)")
	}
	// Deterministic and any World member can derive it.
	if in.Code.Hex() != InboxAudience(w, addr).Code.Hex() {
		t.Error("inbox code not deterministic")
	}
	// Per-address and world-scoped.
	if in.Code.Hex() == InboxAudience(w, derive.Address("0000000000000000000000000000000000000000")).Code.Hex() {
		t.Error("inbox code not address-specific")
	}
	if in.Code.Hex() == InboxAudience(derive.OpenWorld("other"), addr).Code.Hex() {
		t.Error("inbox code not world-scoped")
	}
}

func TestRegistry(t *testing.T) {
	w := derive.OpenWorld(testWorldPhrase)
	reg := NewRegistry()
	reg.Add(PublicAudience(w, 1))
	reg.Add(PublicAudience(w, 2))
	if reg.Len() != 2 {
		t.Fatalf("registry len = %d, want 2", reg.Len())
	}
	pub := PublicAudience(w, 1)
	if a, ok := reg.ByCode([]byte(pub.Code)); !ok || a.ID != pub.ID {
		t.Fatalf("ByCode public-1 = (%+v, %v)", a, ok)
	}
	if _, ok := reg.ByCode([]byte("unknown_code____")); ok {
		t.Error("ByCode returned a match for an unknown code")
	}
}
