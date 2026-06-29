package connections

import (
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/implementations/go/identity"
	"github.com/bpprotocol/blockparty/node/internal/keystore"
)

// fakeNet is an in-memory transport: a block published to an audience is
// delivered (synchronously) to every other manager following that audience —
// the same routing gossipsub provides, minus the network. It also records
// published blocks by ID so tests can inspect them.
type fakeNet struct {
	mu        sync.Mutex
	subs      map[string][]*Manager
	published map[string]*blockpb.Block
}

func newFakeNet() *fakeNet {
	return &fakeNet{subs: map[string][]*Manager{}, published: map[string]*blockpb.Block{}}
}

func (f *fakeNet) pub(m *Manager) Publisher { return &fakePub{net: f, owner: m} }

func (f *fakeNet) block(id string) *blockpb.Block {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.published[id]
}

type fakePub struct {
	net   *fakeNet
	owner *Manager
}

func (p *fakePub) Follow(aud string) error {
	p.net.mu.Lock()
	defer p.net.mu.Unlock()
	for _, m := range p.net.subs[aud] {
		if m == p.owner {
			return nil
		}
	}
	p.net.subs[aud] = append(p.net.subs[aud], p.owner)
	return nil
}

func (p *fakePub) Publish(aud string, b *blockpb.Block) error {
	p.net.mu.Lock()
	subs := append([]*Manager{}, p.net.subs[aud]...)
	if b.Id != nil {
		p.net.published[string(toHex(b.Id))] = b
	}
	p.net.mu.Unlock()
	for _, m := range subs {
		if m != p.owner {
			m.OnBlock(b)
		}
	}
	return nil
}

func toHex(b []byte) []byte {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexdigits[v>>4]
		out[i*2+1] = hexdigits[v&0x0f]
	}
	return out
}

func newManager(t *testing.T, net *fakeNet, seed, pass string) (*Manager, identity.Identity) {
	t.Helper()
	ks, err := keystore.Init(keystore.Path(t.TempDir()), "unlock",
		keystore.Secrets{WorldSeed: seed, IdentityPassphrase: pass})
	if err != nil {
		t.Fatalf("keystore: %v", err)
	}
	m := New(ks, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	m.pub = net.pub(m)
	// Simulate core following the node's own inbox so it receives requests.
	_ = m.pub.Follow(m.InboxCodeHex())
	return m, ks.Identity()
}

func connInfo(t *testing.T, m *Manager, peer derive.Address) ConnInfo {
	t.Helper()
	for _, c := range m.Connections() {
		if c.Peer == string(peer) {
			return c
		}
	}
	t.Fatalf("no connection with %s", peer)
	return ConnInfo{}
}

// TestHandshakeAndPrivateExchange is the #37 acceptance: two personal nodes
// complete a handshake purely via inbox-audience blocks, derive the same private
// audience, and exchange a private-audience block.
func TestHandshakeAndPrivateExchange(t *testing.T) {
	const world = "connections-test-world"
	net := newFakeNet()
	alice, aliceID := newManager(t, net, world, "alice")
	bob, bobID := newManager(t, net, world, "bob")

	// Each node knows the other's identity (resolved from identity blocks IRL).
	alice.AddPeer(bobID.Address, bobID.Kyber.Public, bobID.MLDSA.Public)
	bob.AddPeer(aliceID.Address, aliceID.Kyber.Public, aliceID.MLDSA.Public)

	// Alice initiates; the in-memory transport drives the full handshake.
	if _, err := alice.Start(bobID.Address); err != nil {
		t.Fatalf("Start: %v", err)
	}

	aConn := connInfo(t, alice, bobID.Address)
	bConn := connInfo(t, bob, aliceID.Address)
	if aConn.AudienceCode == "" || aConn.AudienceCode != bConn.AudienceCode {
		t.Fatalf("private audiences differ: alice=%q bob=%q", aConn.AudienceCode, bConn.AudienceCode)
	}

	// Exchange one private-audience block.
	id, err := alice.SendText(bobID.Address, "hello over the private channel")
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}
	got, ok := bob.OpenPrivatePost(net.block(id))
	if !ok {
		t.Fatal("bob could not open the private post")
	}
	if got != "hello over the private channel" {
		t.Errorf("private text = %q", got)
	}
}

func TestRotateAdvancesBothEpochs(t *testing.T) {
	const world = "connections-rotate-world"
	net := newFakeNet()
	alice, aliceID := newManager(t, net, world, "alice")
	bob, bobID := newManager(t, net, world, "bob")
	alice.AddPeer(bobID.Address, bobID.Kyber.Public, bobID.MLDSA.Public)
	bob.AddPeer(aliceID.Address, aliceID.Kyber.Public, aliceID.MLDSA.Public)
	if _, err := alice.Start(bobID.Address); err != nil {
		t.Fatalf("Start: %v", err)
	}
	before := connInfo(t, alice, bobID.Address).AudienceCode

	if err := alice.Rotate(bobID.Address); err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	aConn := connInfo(t, alice, bobID.Address)
	bConn := connInfo(t, bob, aliceID.Address)
	if aConn.Epoch != 1 || bConn.Epoch != 1 {
		t.Fatalf("epochs after rotate: alice=%d bob=%d, want 1/1", aConn.Epoch, bConn.Epoch)
	}
	if aConn.AudienceCode == before {
		t.Error("audience code did not change after rotation")
	}
	if aConn.AudienceCode != bConn.AudienceCode {
		t.Errorf("post-rotate audiences differ: alice=%q bob=%q", aConn.AudienceCode, bConn.AudienceCode)
	}

	// The new-epoch private channel still works.
	id, err := alice.SendText(bobID.Address, "after rotation")
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}
	if got, ok := bob.OpenPrivatePost(net.block(id)); !ok || got != "after rotation" {
		t.Errorf("post-rotate exchange failed: got=%q ok=%v", got, ok)
	}
}

func TestCloseDropsConnection(t *testing.T) {
	const world = "connections-close-world"
	net := newFakeNet()
	alice, aliceID := newManager(t, net, world, "alice")
	bob, bobID := newManager(t, net, world, "bob")
	alice.AddPeer(bobID.Address, bobID.Kyber.Public, bobID.MLDSA.Public)
	bob.AddPeer(aliceID.Address, aliceID.Kyber.Public, aliceID.MLDSA.Public)
	if _, err := alice.Start(bobID.Address); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := alice.Close(bobID.Address); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if len(alice.Connections()) != 0 {
		t.Error("alice still has a connection after close")
	}
	if len(bob.Connections()) != 0 {
		t.Error("bob still has a connection after peer close")
	}
}
