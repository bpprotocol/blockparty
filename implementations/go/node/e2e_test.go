package node

import (
	"testing"

	"github.com/bpprotocol/blockparty/implementations/go/audiences"
	"github.com/bpprotocol/blockparty/implementations/go/block"
	"github.com/bpprotocol/blockparty/implementations/go/blocktypes"
	"github.com/bpprotocol/blockparty/implementations/go/connections"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/implementations/go/encryption"
	"github.com/bpprotocol/blockparty/implementations/go/identity"
)

const (
	worldSeed = "bpprotocol.org/v1/e2e-world"
	ts        = int64(1700000000)
)

// TestPublicAudienceExchange: two instances sharing a World exchange an
// encrypted block via a public audience over the filesystem transport.
func TestPublicAudienceExchange(t *testing.T) {
	w := derive.OpenWorld(worldSeed)
	alice := identity.OpenIdentity(w, "alice")
	net := FileStore{Dir: t.TempDir()}
	pub := audiences.PublicAudience(w, 1)

	// Alice writes an encrypted post to public-1.
	b, err := BuildPost(w, alice, pub.Code, pub.Secret, ts, "nothing can stop the signal")
	if err != nil {
		t.Fatalf("BuildPost: %v", err)
	}
	if _, err := net.Write(b); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Bob (another World member) reads, verifies world membership, and decrypts.
	r := blocktypes.NewResolver(w)
	bobPub := audiences.PublicAudience(w, 1) // independently derived
	blocks, err := net.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	var got string
	var found bool
	for _, blk := range blocks {
		ok, err := OpenPost(w, r, blk, bobPub.Secret, &got)
		if err != nil {
			t.Fatalf("OpenPost: %v", err)
		}
		if ok {
			found = true
		}
	}
	if !found || got != "nothing can stop the signal" {
		t.Fatalf("Bob read %q (found=%v)", got, found)
	}
}

// TestConnectionExchange: two instances complete the connect.* handshake and
// exchange an encrypted block over the resulting private audience, via the
// filesystem transport. This is the end-to-end "connect and exchange" proof.
func TestConnectionExchange(t *testing.T) {
	w := derive.OpenWorld(worldSeed)
	alice := identity.OpenIdentity(w, "alice")
	bob := identity.OpenIdentity(w, "bob")
	alicePeer := connections.Peer{Address: alice.Address, KyberPub: alice.Kyber.Public}
	bobPeer := connections.Peer{Address: bob.Address, KyberPub: bob.Kyber.Public}
	net := FileStore{Dir: t.TempDir()}

	// Handshake (request/response would themselves travel over the transport;
	// here we pass the payloads directly for brevity — the wire form is the same).
	pending, req, err := connections.StartRequest(alice, bobPeer)
	if err != nil {
		t.Fatalf("StartRequest: %v", err)
	}
	resp, bobConn, err := connections.AcceptRequest(bob, alicePeer, req, "req-id")
	if err != nil {
		t.Fatalf("AcceptRequest: %v", err)
	}
	aliceConn, err := pending.Complete(resp)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Alice posts to the private audience; the block goes over the transport.
	b, err := BuildPost(w, alice, aliceConn.AudienceCode(), aliceConn.AudienceSecret(), ts, "private hello")
	if err != nil {
		t.Fatalf("BuildPost: %v", err)
	}
	if _, err := net.Write(b); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Bob reads, fully verifies Alice's authorship (he knows her identity),
	// and decrypts with his independently-derived connection secret.
	blocks, err := net.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	got := blocks[0]
	if err := block.Verify(got, w, alice.MLDSA.Public); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	r := blocktypes.NewResolver(w)
	var text string
	ok, err := OpenPost(w, r, got, bobConn.AudienceSecret(), &text)
	if err != nil || !ok {
		t.Fatalf("OpenPost: ok=%v err=%v", ok, err)
	}
	if text != "private hello" {
		t.Fatalf("Bob decrypted %q, want %q", text, "private hello")
	}
}

// TestOutsiderCannotDecrypt confirms a same-transport block is opaque to a
// different World.
func TestOutsiderCannotDecrypt(t *testing.T) {
	w := derive.OpenWorld(worldSeed)
	alice := identity.OpenIdentity(w, "alice")
	pub := audiences.PublicAudience(w, 1)
	b, _ := BuildPost(w, alice, pub.Code, pub.Secret, ts, "secret")

	outsider := derive.OpenWorld("outsider")
	wrong := audiences.PublicAudience(outsider, 1)
	if _, err := encryption.DecryptData(wrong.Secret, b); err != encryption.ErrDecrypt {
		t.Fatalf("outsider decrypt = %v, want ErrDecrypt", err)
	}
}
