// Package connections drives the connect.* handshake for a personal-mode node
// (#37), letting the headless daemon establish and maintain private channels
// while the UI is closed.
//
// Handshake blocks are exchanged over inbox audiences (plaintext — KEM
// ciphertexts are safe to publish): a connect.request lands on the target's
// inbox, the target replies with a connect.response on the initiator's inbox,
// and both derive the same private audience. Subsequent connect.rotate /
// connect.close (and application messages) flow encrypted on the private
// audience. The actual cryptography lives in the SDK connections package; this
// manager wires it to the node's transport (gossip) and store (via the guard's
// accept hook).
package connections

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/cloudflare/circl/kem"
	"github.com/cloudflare/circl/sign"
	"google.golang.org/protobuf/proto"

	"github.com/bpprotocol/blockparty/implementations/go/audiences"
	"github.com/bpprotocol/blockparty/implementations/go/block"
	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
	"github.com/bpprotocol/blockparty/implementations/go/blocktypes"
	sdkconn "github.com/bpprotocol/blockparty/implementations/go/connections"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/implementations/go/encryption"
	bpnode "github.com/bpprotocol/blockparty/implementations/go/node"
	"github.com/bpprotocol/blockparty/node/internal/keystore"
)

// replayWindow bounds handshake timestamp acceptance (connections.md default).
const replayWindow = 5 * time.Minute

// ErrNoConnection is returned when no connection exists for a peer.
var ErrNoConnection = errors.New("connections: no connection with peer")

// Publisher is the transport the manager uses: publish blocks to an audience
// topic and (un)follow audience topics. Implemented by *gossip.Gossip.
type Publisher interface {
	Publish(audienceHex string, b *blockpb.Block) error
	Follow(audienceHex string) error
}

// ConnInfo is a connection summary for the API/status.
type ConnInfo struct {
	Peer         string `json:"peer"`
	Epoch        uint64 `json:"epoch"`
	AudienceCode string `json:"audience_code"`
}

type peerEntry struct {
	peer  sdkconn.Peer
	mldsa sign.PublicKey
}

// Manager owns a personal node's connections.
type Manager struct {
	ks       *keystore.Keystore
	pub      Publisher
	resolver *blocktypes.Resolver
	replay   *sdkconn.ReplayGuard
	log      *slog.Logger
	now      func() int64
	inbox    derive.Code

	mu      sync.Mutex
	peers   map[string]peerEntry               // peer address → known keys
	pending map[string]*sdkconn.PendingRequest // request block id → pending (initiator)
	conns   map[string]*sdkconn.Connection     // peer address → connection
	byAud   map[string]string                  // private audience hex → peer address
}

// New builds a manager for the keystore's identity, publishing via pub.
func New(ks *keystore.Keystore, pub Publisher, log *slog.Logger) *Manager {
	w := ks.World()
	self := ks.Identity()
	return &Manager{
		ks:       ks,
		pub:      pub,
		resolver: blocktypes.NewResolver(w),
		replay:   sdkconn.NewReplayGuard(replayWindow),
		log:      log,
		now:      func() int64 { return time.Now().Unix() },
		inbox:    audiences.InboxAudience(w, self.Address).Code,
		peers:    make(map[string]peerEntry),
		pending:  make(map[string]*sdkconn.PendingRequest),
		conns:    make(map[string]*sdkconn.Connection),
		byAud:    make(map[string]string),
	}
}

// InboxCodeHex is the node's inbox audience code (the topic peers send
// handshake requests to). The node must follow it to receive requests.
func (m *Manager) InboxCodeHex() string { return m.inbox.Hex() }

// AddPeer registers a known peer so the node can connect to it and resolve it as
// the author of an incoming request. Populated from identity blocks by callers.
func (m *Manager) AddPeer(addr derive.Address, kyberPub kem.PublicKey, mldsaPub sign.PublicKey) {
	m.mu.Lock()
	m.peers[string(addr)] = peerEntry{peer: sdkconn.Peer{Address: addr, KyberPub: kyberPub}, mldsa: mldsaPub}
	m.mu.Unlock()
}

// Start initiates a connection to a known peer, publishing a connect.request to
// its inbox. Returns the request block ID used to correlate the response.
func (m *Manager) Start(addr derive.Address) (string, error) {
	m.mu.Lock()
	entry, ok := m.peers[string(addr)]
	m.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("connections: unknown peer %s (register it first)", addr)
	}

	pending, req, err := sdkconn.StartRequest(m.ks.Identity(), entry.peer)
	if err != nil {
		return "", fmt.Errorf("connections: start request: %w", err)
	}
	targetInbox := audiences.InboxAudience(m.ks.World(), addr).Code
	b, err := m.buildPlain(blocktypes.TypeConnectRequest, targetInbox, req)
	if err != nil {
		return "", err
	}
	reqID := block.IDHex(b)
	m.mu.Lock()
	m.pending[reqID] = pending
	m.mu.Unlock()

	_ = m.pub.Follow(targetInbox.Hex())
	if err := m.pub.Publish(targetInbox.Hex(), b); err != nil {
		return "", fmt.Errorf("connections: publish request: %w", err)
	}
	m.log.Info("connect request sent", "peer", addr)
	return reqID, nil
}

// OnBlock handles an accepted block that may be part of a handshake. It is wired
// to the guard's accept hook, so it sees every validated block.
func (m *Manager) OnBlock(b *blockpb.Block) {
	urn, ok := m.resolver.TypeURN(b.TypeCode)
	if !ok {
		return
	}
	switch urn {
	case blocktypes.TypeConnectRequest:
		m.handleRequest(b)
	case blocktypes.TypeConnectResponse:
		m.handleResponse(b)
	case blocktypes.TypeConnectRotate, blocktypes.TypeConnectClose:
		m.handlePrivate(b, urn)
	}
}

func (m *Manager) handleRequest(b *blockpb.Block) {
	msg, err := blocktypes.DecodePayload(blocktypes.TypeConnectRequest, b.Data)
	if err != nil {
		return
	}
	req := msg.(*blockpb.ConnectRequest)
	self := m.ks.Identity()
	if req.Target != string(self.Address) {
		return // not addressed to us
	}
	if err := m.replay.Check(self.Address, req.Nonce, b.Timestamp); err != nil {
		m.log.Debug("connections: replay/stale request dropped", "err", err)
		return
	}
	initiator, ok := m.resolveAuthor(b)
	if !ok {
		m.log.Debug("connections: connect.request from unknown initiator; ignoring")
		return
	}

	resp, conn, err := sdkconn.AcceptRequest(self, initiator, req, block.IDHex(b))
	if err != nil {
		m.log.Debug("connections: accept failed", "err", err)
		return
	}
	m.register(conn)

	initInbox := audiences.InboxAudience(m.ks.World(), initiator.Address).Code
	rb, err := m.buildPlain(blocktypes.TypeConnectResponse, initInbox, resp)
	if err != nil {
		return
	}
	_ = m.pub.Follow(initInbox.Hex())
	if err := m.pub.Publish(initInbox.Hex(), rb); err != nil {
		m.log.Debug("connections: publish response failed", "err", err)
		return
	}
	m.log.Info("connection accepted", "peer", initiator.Address)
}

func (m *Manager) handleResponse(b *blockpb.Block) {
	msg, err := blocktypes.DecodePayload(blocktypes.TypeConnectResponse, b.Data)
	if err != nil {
		return
	}
	resp := msg.(*blockpb.ConnectResponse)

	m.mu.Lock()
	pending, ok := m.pending[resp.Request]
	if ok {
		delete(m.pending, resp.Request)
	}
	m.mu.Unlock()
	if !ok {
		return // not our pending request
	}
	conn, err := pending.Complete(resp)
	if err != nil {
		m.log.Debug("connections: complete failed", "err", err)
		return
	}
	m.register(conn)
	m.log.Info("connection established", "peer", conn.PeerAddress())
}

// handlePrivate handles connect.rotate / connect.close blocks on a private
// audience (encrypted with the connection's current secret).
func (m *Manager) handlePrivate(b *blockpb.Block, urn string) {
	audHex := hex.EncodeToString(b.AudienceCode)
	m.mu.Lock()
	peerAddr, ok := m.byAud[audHex]
	conn := m.conns[peerAddr]
	m.mu.Unlock()
	if !ok || conn == nil {
		return
	}
	plain, err := encryption.DecryptData(conn.AudienceSecret(), b)
	if err != nil {
		return
	}
	msg, err := blocktypes.DecodePayload(urn, plain)
	if err != nil {
		return
	}
	switch urn {
	case blocktypes.TypeConnectRotate:
		rot := msg.(*blockpb.ConnectRotate)
		if err := conn.ApplyRotate(rot); err != nil {
			m.log.Debug("connections: apply rotate failed", "err", err)
			return
		}
		m.reindex(peerAddr, audHex, conn)
		m.log.Info("connection rotated", "peer", peerAddr, "epoch", conn.Epoch())
	case blocktypes.TypeConnectClose:
		m.drop(peerAddr, audHex)
		m.log.Info("connection closed by peer", "peer", peerAddr)
	}
}

// SendText posts a content.post encrypted to a peer's private audience.
func (m *Manager) SendText(addr derive.Address, text string) (string, error) {
	m.mu.Lock()
	conn, ok := m.conns[string(addr)]
	m.mu.Unlock()
	if !ok {
		return "", ErrNoConnection
	}
	b, err := bpnode.BuildPost(m.ks.World(), m.ks.Identity(), conn.AudienceCode(), conn.AudienceSecret(), m.now(), text)
	if err != nil {
		return "", err
	}
	if err := m.pub.Publish(conn.AudienceCode().Hex(), b); err != nil {
		return "", err
	}
	return block.IDHex(b), nil
}

// Rotate advances a connection to a new epoch and notifies the peer.
func (m *Manager) Rotate(addr derive.Address) error {
	m.mu.Lock()
	conn, ok := m.conns[string(addr)]
	m.mu.Unlock()
	if !ok {
		return ErrNoConnection
	}
	oldCode := conn.AudienceCode()
	oldSecret := conn.AudienceSecret()
	oldHex := oldCode.Hex()

	rot, err := conn.Rotate()
	if err != nil {
		return err
	}
	b, err := m.buildEncrypted(blocktypes.TypeConnectRotate, oldCode, oldSecret, rot)
	if err != nil {
		return err
	}
	if err := m.pub.Publish(oldHex, b); err != nil {
		return err
	}
	m.reindex(string(addr), oldHex, conn)
	return nil
}

// Close tears down a connection and notifies the peer.
func (m *Manager) Close(addr derive.Address) error {
	m.mu.Lock()
	conn, ok := m.conns[string(addr)]
	m.mu.Unlock()
	if !ok {
		return ErrNoConnection
	}
	code := conn.AudienceCode()
	b, err := m.buildEncrypted(blocktypes.TypeConnectClose, code, conn.AudienceSecret(), sdkconn.BuildClose(addr, "closed"))
	if err == nil {
		_ = m.pub.Publish(code.Hex(), b)
	}
	m.drop(string(addr), code.Hex())
	return nil
}

// OpenPrivatePost decrypts a content.post on one of this node's private
// connection audiences, returning its text. ok is false if the block is not on a
// known private audience or is not a readable content.post.
func (m *Manager) OpenPrivatePost(b *blockpb.Block) (string, bool) {
	audHex := hex.EncodeToString(b.AudienceCode)
	m.mu.Lock()
	peerAddr, ok := m.byAud[audHex]
	conn := m.conns[peerAddr]
	m.mu.Unlock()
	if !ok || conn == nil {
		return "", false
	}
	var text string
	opened, err := bpnode.OpenPost(m.ks.World(), m.resolver, b, conn.AudienceSecret(), &text)
	if err != nil || !opened {
		return "", false
	}
	return text, true
}

// Connections lists the active connections.
func (m *Manager) Connections() []ConnInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]ConnInfo, 0, len(m.conns))
	for addr, c := range m.conns {
		out = append(out, ConnInfo{Peer: addr, Epoch: c.Epoch(), AudienceCode: c.AudienceCode().Hex()})
	}
	return out
}

// --- internals ---

func (m *Manager) register(conn *sdkconn.Connection) {
	audHex := conn.AudienceCode().Hex()
	m.mu.Lock()
	m.conns[string(conn.PeerAddress())] = conn
	m.byAud[audHex] = string(conn.PeerAddress())
	m.mu.Unlock()
	_ = m.pub.Follow(audHex)
}

// reindex updates the audience→peer mapping after a connection's audience code
// changes (rotation) and follows the new private audience.
func (m *Manager) reindex(peerAddr, oldAudHex string, conn *sdkconn.Connection) {
	newHex := conn.AudienceCode().Hex()
	m.mu.Lock()
	delete(m.byAud, oldAudHex)
	m.byAud[newHex] = peerAddr
	m.mu.Unlock()
	_ = m.pub.Follow(newHex)
}

func (m *Manager) drop(peerAddr, audHex string) {
	m.mu.Lock()
	delete(m.conns, peerAddr)
	delete(m.byAud, audHex)
	m.mu.Unlock()
}

func (m *Manager) resolveAuthor(b *blockpb.Block) (sdkconn.Peer, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.peers {
		if block.VerifyAuthor(b, e.mldsa) == nil {
			return e.peer, true
		}
	}
	return sdkconn.Peer{}, false
}

// buildPlain wraps a payload in a signed plaintext block on an audience (used
// for the inbox handshake blocks, which carry no secrets).
func (m *Manager) buildPlain(typeURN string, audCode derive.Code, payload proto.Message) (*blockpb.Block, error) {
	data, err := blocktypes.MarshalPayload(payload)
	if err != nil {
		return nil, err
	}
	return m.sign(typeURN, audCode, data), nil
}

// buildEncrypted wraps a payload in a signed block AEAD-encrypted to an audience
// (used for connect.rotate / connect.close on the private audience).
func (m *Manager) buildEncrypted(typeURN string, audCode derive.Code, audSecret []byte, payload proto.Message) (*blockpb.Block, error) {
	plain, err := blocktypes.MarshalPayload(payload)
	if err != nil {
		return nil, err
	}
	w := m.ks.World()
	self := m.ks.Identity()
	ts := m.now()
	typeCode := derive.GetTypeCode(w, typeURN)
	data, err := encryption.EncryptForBlock(audSecret, block.Version, []byte(typeCode), []byte(audCode), ts, plain)
	if err != nil {
		return nil, err
	}
	b := block.New(self.Address, typeCode, audCode, ts, data)
	block.Sign(b, w, self.MLDSA)
	return b, nil
}

func (m *Manager) sign(typeURN string, audCode derive.Code, data []byte) *blockpb.Block {
	w := m.ks.World()
	self := m.ks.Identity()
	typeCode := derive.GetTypeCode(w, typeURN)
	b := block.New(self.Address, typeCode, audCode, m.now(), data)
	block.Sign(b, w, self.MLDSA)
	return b
}
