package connections

import (
	"bytes"
	"testing"
	"time"

	"github.com/bpprotocol/blockparty/implementations/go/block"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/implementations/go/encryption"
	"github.com/bpprotocol/blockparty/implementations/go/identity"
)

const (
	testWorldPhrase = "bpprotocol.org/v1/test-world"
	connectType     = "bpprotocol.org/v1/types/connect.identity"
	testTimestamp   = int64(1700000000)
)

type parties struct {
	world     derive.World
	alice     identity.Identity
	bob       identity.Identity
	alicePeer Peer
	bobPeer   Peer
}

func setup() parties {
	w := derive.OpenWorld(testWorldPhrase)
	alice := identity.OpenIdentity(w, "alice")
	bob := identity.OpenIdentity(w, "bob")
	return parties{
		world:     w,
		alice:     alice,
		bob:       bob,
		alicePeer: Peer{Address: alice.Address, KyberPub: alice.Kyber.Public},
		bobPeer:   Peer{Address: bob.Address, KyberPub: bob.Kyber.Public},
	}
}

// handshake runs a full Alice->Bob handshake and returns both connections.
func (p parties) handshake(t *testing.T) (alice, bob *Connection) {
	t.Helper()
	pending, req, err := StartRequest(p.alice, p.bobPeer)
	if err != nil {
		t.Fatalf("StartRequest: %v", err)
	}
	resp, bobConn, err := AcceptRequest(p.bob, p.alicePeer, req, "request-block-id")
	if err != nil {
		t.Fatalf("AcceptRequest: %v", err)
	}
	aliceConn, err := pending.Complete(resp)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	return aliceConn, bobConn
}

func TestHandshakeAgreement(t *testing.T) {
	p := setup()
	a, b := p.handshake(t)

	if !bytes.Equal(a.AudienceSecret(), b.AudienceSecret()) {
		t.Fatal("parties derived different audience secrets")
	}
	if a.AudienceCode().Hex() != b.AudienceCode().Hex() {
		t.Fatal("parties derived different audience codes")
	}
	if a.Epoch() != 0 || b.Epoch() != 0 {
		t.Fatalf("epochs = %d/%d, want 0/0", a.Epoch(), b.Epoch())
	}
	if a.PeerAddress() != p.bob.Address || b.PeerAddress() != p.alice.Address {
		t.Fatal("peer addresses not set correctly")
	}
}

func TestPrivateAudienceRoundTrip(t *testing.T) {
	p := setup()
	a, b := p.handshake(t)

	typeCode := derive.GetTypeCode(p.world, connectType)
	code := a.AudienceCode()
	plaintext := []byte(`{"name":"Alice"}`)

	data, err := encryption.EncryptForBlock(a.AudienceSecret(), block.Version, []byte(typeCode), []byte(code), testTimestamp, plaintext)
	if err != nil {
		t.Fatalf("EncryptForBlock: %v", err)
	}
	blk := block.New(p.alice.Address, typeCode, code, testTimestamp, data)
	block.Sign(blk, p.world, p.alice.MLDSA)

	// Bob decrypts with his independently-derived audience secret.
	got, err := encryption.DecryptData(b.AudienceSecret(), blk)
	if err != nil {
		t.Fatalf("Bob DecryptData: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("decrypted %q, want %q", got, plaintext)
	}
}

func TestAcceptWrongTarget(t *testing.T) {
	p := setup()
	_, req, err := StartRequest(p.alice, p.bobPeer)
	if err != nil {
		t.Fatalf("StartRequest: %v", err)
	}
	req.Target = string(p.alice.Address) // not Bob
	if _, _, err := AcceptRequest(p.bob, p.alicePeer, req, "id"); err != ErrWrongTarget {
		t.Fatalf("AcceptRequest wrong target = %v, want ErrWrongTarget", err)
	}
}

func TestRotation(t *testing.T) {
	p := setup()
	a, b := p.handshake(t)

	s0 := append([]byte(nil), a.AudienceSecret()...)
	code0 := a.AudienceCode().Hex()

	rot, err := a.Rotate()
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if err := b.ApplyRotate(rot); err != nil {
		t.Fatalf("ApplyRotate: %v", err)
	}

	if a.Epoch() != 1 || b.Epoch() != 1 {
		t.Fatalf("epochs after rotate = %d/%d, want 1/1", a.Epoch(), b.Epoch())
	}
	if !bytes.Equal(a.AudienceSecret(), b.AudienceSecret()) {
		t.Fatal("parties disagree after rotation")
	}
	if bytes.Equal(a.AudienceSecret(), s0) {
		t.Fatal("rotation did not produce a fresh secret")
	}
	if a.AudienceCode().Hex() == code0 {
		t.Fatal("rotation did not produce a fresh audience code")
	}
	if a.AudienceCode().Hex() != rot.NewChannel {
		t.Fatal("rotator's new code disagrees with the payload's new_channel")
	}
}

func TestApplyRotateMismatch(t *testing.T) {
	p := setup()
	a, b := p.handshake(t)
	rot, err := a.Rotate()
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	rot.NewChannel = "deadbeefdeadbeefdeadbeefdeadbeef"
	if err := b.ApplyRotate(rot); err != ErrRotateMismatch {
		t.Fatalf("ApplyRotate mismatch = %v, want ErrRotateMismatch", err)
	}
}

func TestReplayGuard(t *testing.T) {
	g := NewReplayGuard(DefaultReplayWindow)
	g.now = func() time.Time { return time.Unix(testTimestamp, 0) }
	target := derive.Address("cff62cff35a0cc4271c262c07a86cbccc63ad13a")
	nonce := bytes.Repeat([]byte{0xab}, NonceSize)

	if err := g.Check(target, nonce, testTimestamp); err != nil {
		t.Fatalf("first Check: %v", err)
	}
	if err := g.Check(target, nonce, testTimestamp); err != ErrReplay {
		t.Fatalf("replayed Check = %v, want ErrReplay", err)
	}
	// A fresh nonce but a stale timestamp.
	fresh := bytes.Repeat([]byte{0xcd}, NonceSize)
	if err := g.Check(target, fresh, testTimestamp-1000); err != ErrStale {
		t.Fatalf("stale Check = %v, want ErrStale", err)
	}
	// Same nonce but a different target is independent.
	if err := g.Check(derive.Address("0000000000000000000000000000000000000000"), nonce, testTimestamp); err != nil {
		t.Fatalf("different-target Check: %v", err)
	}
}

func TestBuildClose(t *testing.T) {
	c := BuildClose(derive.Address("abc"), "rotated_keys")
	if c.Target != "abc" || c.Reason != "rotated_keys" {
		t.Fatalf("unexpected close payload: %+v", c)
	}
}
