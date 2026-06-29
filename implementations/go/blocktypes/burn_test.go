package blocktypes

import (
	"testing"

	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/implementations/go/identity"
)

func TestVerifyBurnGenuine(t *testing.T) {
	w := derive.OpenWorld(testWorldPhrase)
	id := identity.OpenIdentity(w, "to-be-burned")

	burn, err := BuildBurn(id, "voluntary")
	if err != nil {
		t.Fatalf("BuildBurn: %v", err)
	}
	if burn.Identity != string(id.Address) {
		t.Fatalf("burn identity %q != %q", burn.Identity, id.Address)
	}
	ok, err := VerifyBurn(burn)
	if err != nil {
		t.Fatalf("VerifyBurn: %v", err)
	}
	if !ok {
		t.Fatal("genuine burn failed verification")
	}
}

func TestVerifyBurnWrongAddress(t *testing.T) {
	w := derive.OpenWorld(testWorldPhrase)
	id := identity.OpenIdentity(w, "victim")
	burn, _ := BuildBurn(id, "voluntary")
	// Claim a different identity than the revealed keys derive.
	burn.Identity = "0000000000000000000000000000000000000000"

	ok, err := VerifyBurn(burn)
	if err != nil {
		t.Fatalf("VerifyBurn: %v", err)
	}
	if ok {
		t.Fatal("burn with mismatched address verified")
	}
}

func TestTrustStateFlagsBurned(t *testing.T) {
	w := derive.OpenWorld(testWorldPhrase)
	alice := identity.OpenIdentity(w, "alice")
	bob := identity.OpenIdentity(w, "bob")

	ts := NewTrustState()
	if ts.IsBurned(string(alice.Address)) {
		t.Fatal("identity burned before any burn recorded")
	}

	burn, _ := BuildBurn(alice, "compromised")
	genuine, err := ts.RecordBurn(burn)
	if err != nil || !genuine {
		t.Fatalf("RecordBurn = (%v, %v)", genuine, err)
	}
	if !ts.IsBurned(string(alice.Address)) {
		t.Fatal("alice not flagged as burned after a genuine burn")
	}
	if ts.IsBurned(string(bob.Address)) {
		t.Fatal("bob wrongly flagged as burned")
	}

	// A forged burn (mismatched address) must not flip trust state.
	forged, _ := BuildBurn(bob, "voluntary")
	forged.Identity = string(alice.Address) // lie: claim it burns alice
	genuine, err = ts.RecordBurn(forged)
	if err != nil {
		t.Fatalf("RecordBurn forged: %v", err)
	}
	if genuine {
		t.Fatal("forged burn reported as genuine")
	}
}

func TestRecordBurnBadKeyBytes(t *testing.T) {
	ts := NewTrustState()
	bad := &blockpb.IdentityBurn{Identity: "addr", RevealedMlDsa: []byte("nonsense"), RevealedKyber: []byte("nonsense")}
	if genuine, _ := ts.RecordBurn(bad); genuine {
		t.Fatal("burn with malformed keys reported genuine")
	}
}
