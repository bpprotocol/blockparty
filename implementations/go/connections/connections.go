package connections

import (
	"crypto/rand"
	"errors"

	"github.com/cloudflare/circl/kem"

	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
	"github.com/bpprotocol/blockparty/implementations/go/crypto"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/implementations/go/identity"
)

// NonceSize is the length of the handshake nonces.
const NonceSize = 32

var (
	// ErrWrongTarget is returned when a request's target is not this identity.
	ErrWrongTarget = errors.New("connections: request target is not this identity")
	// ErrRotateMismatch is returned when a rotation's new_channel does not match
	// the recipient's recomputed audience code.
	ErrRotateMismatch = errors.New("connections: rotation channel mismatch")
)

// Peer is a remote identity addressable for a connection: its address and
// static KEM public key (learned from its identity block).
type Peer struct {
	Address  derive.Address
	KyberPub kem.PublicKey
}

// Connection is an established connection at a given epoch. Its audience secret
// and code drive the private channel's encryption (see the encryption package).
type Connection struct {
	self      identity.Identity
	peerAddr  derive.Address
	peerKyber kem.PublicKey
	epoch     uint64
	secret    []byte // connectionSecret(epoch)
}

// Epoch returns the connection's current epoch.
func (c *Connection) Epoch() uint64 { return c.epoch }

// PeerAddress returns the connected peer's address.
func (c *Connection) PeerAddress() derive.Address { return c.peerAddr }

// AudienceSecret returns the current epoch's private audience secret.
func (c *Connection) AudienceSecret() []byte { return audienceSecret(c.secret, c.epoch) }

// AudienceCode returns the current epoch's private audience code.
func (c *Connection) AudienceCode() derive.Code { return audienceCode(c.secret, c.epoch) }

// PendingRequest holds the initiator's state between sending connect.request
// and receiving connect.response.
type PendingRequest struct {
	self  identity.Identity
	peer  Peer
	eph   crypto.KyberKeyPair
	ss1   []byte
	nonce []byte
}

// StartRequest builds a connect.request to peer: a fresh ephemeral KEM key for
// forward secrecy, a shared secret encapsulated to the peer's static key, and a
// random nonce. The returned PendingRequest is completed with Complete once the
// response arrives.
func StartRequest(self identity.Identity, peer Peer) (*PendingRequest, *blockpb.ConnectRequest, error) {
	eph := crypto.MakeKyberPair(randomBytes(32))
	ct1, ss1, err := crypto.Encapsulate(peer.KyberPub)
	if err != nil {
		return nil, nil, err
	}
	nonce := randomBytes(NonceSize)
	req := &blockpb.ConnectRequest{
		Target:        string(peer.Address),
		InitKemPub:    eph.PublicBytes(),
		KemCiphertext: ct1,
		Nonce:         nonce,
	}
	return &PendingRequest{self: self, peer: peer, eph: eph, ss1: ss1, nonce: nonce}, req, nil
}

// AcceptRequest consumes a connect.request addressed to self from initiator,
// returning the connect.response payload and the established Connection.
// requestBlockID is the block ID of the request (carried in the response for
// correlation). The caller resolves the initiator's Peer (address + static KEM
// key) from the request's author before calling.
func AcceptRequest(self identity.Identity, initiator Peer, req *blockpb.ConnectRequest, requestBlockID string) (*blockpb.ConnectResponse, *Connection, error) {
	if req.Target != string(self.Address) {
		return nil, nil, ErrWrongTarget
	}
	ss1, err := self.Kyber.Decapsulate(req.KemCiphertext)
	if err != nil {
		return nil, nil, err
	}
	ephPub, err := crypto.KEMScheme().UnmarshalBinaryPublicKey(req.InitKemPub)
	if err != nil {
		return nil, nil, err
	}
	ct2, ss2, err := crypto.Encapsulate(ephPub)
	if err != nil {
		return nil, nil, err
	}
	nonceResp := randomBytes(NonceSize)
	conn := &Connection{
		self:      self,
		peerAddr:  initiator.Address,
		peerKyber: initiator.KyberPub,
		epoch:     0,
		secret:    connectionSecret0(ss1, ss2, req.Nonce, nonceResp),
	}
	resp := &blockpb.ConnectResponse{
		Request:       requestBlockID,
		KemCiphertext: ct2,
		NonceResponse: nonceResp,
	}
	return resp, conn, nil
}

// Complete finishes the initiator's handshake from the connect.response,
// recovering the second shared secret and deriving the Connection.
func (p *PendingRequest) Complete(resp *blockpb.ConnectResponse) (*Connection, error) {
	ss2, err := p.eph.Decapsulate(resp.KemCiphertext)
	if err != nil {
		return nil, err
	}
	return &Connection{
		self:      p.self,
		peerAddr:  p.peer.Address,
		peerKyber: p.peer.KyberPub,
		epoch:     0,
		secret:    connectionSecret0(p.ss1, ss2, p.nonce, resp.NonceResponse),
	}, nil
}

// Rotate advances this connection to the next epoch with fresh KEM entropy and
// returns the connect.rotate payload (to be sent encrypted to the current
// audience). The connection is mutated to the new epoch on success.
func (c *Connection) Rotate() (*blockpb.ConnectRotate, error) {
	ct3, ss3, err := crypto.Encapsulate(c.peerKyber)
	if err != nil {
		return nil, err
	}
	c.secret = rotatedSecret(c.secret, ss3, ct3, c.epoch+1)
	c.epoch++
	return &blockpb.ConnectRotate{NewChannel: c.AudienceCode().Hex(), KemCiphertext: ct3}, nil
}

// ApplyRotate advances this connection in response to a peer's connect.rotate,
// recovering the fresh entropy and verifying that the recomputed audience code
// matches the payload's new_channel before committing.
func (c *Connection) ApplyRotate(rot *blockpb.ConnectRotate) error {
	ss3, err := c.self.Kyber.Decapsulate(rot.KemCiphertext)
	if err != nil {
		return err
	}
	next := rotatedSecret(c.secret, ss3, rot.KemCiphertext, c.epoch+1)
	if audienceCode(next, c.epoch+1).Hex() != rot.NewChannel {
		return ErrRotateMismatch
	}
	c.secret = next
	c.epoch++
	return nil
}

// BuildClose builds a connect.close payload notifying peer of teardown.
func BuildClose(peer derive.Address, reason string) *blockpb.ConnectClose {
	return &blockpb.ConnectClose{Target: string(peer), Reason: reason}
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("connections: out of randomness: " + err.Error())
	}
	return b
}
