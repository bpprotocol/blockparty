// Package connections implements the BlockParty connection handshake (issue
// #7), following protocol/specs/connections.md.
//
// A connection is a private channel between two identities established by a
// two-message ML-KEM768 handshake, yielding a private audience whose secret
// only the two parties can derive:
//
//	Alice: pending, req  := StartRequest(alice, bobPeer)        // connect.request
//	Bob:   resp, bobConn := AcceptRequest(bob, alicePeer, req)  // connect.response
//	Alice: aliceConn     := pending.Complete(resp)
//	// bobConn.AudienceSecret() == aliceConn.AudienceSecret()
//
// The initiator contributes a fresh ephemeral KEM key (forward secrecy) and
// encapsulates to the responder's static key; the responder encapsulates back
// to the ephemeral key. Both shared secrets and both nonces are mixed into the
// connection secret, from which per-epoch audience secrets and codes are
// derived (see encryption package #5 for the AEAD that consumes them).
//
// Rotate / ApplyRotate advance the channel to a new epoch with fresh KEM
// entropy (a forward ratchet); BuildClose tears it down. ReplayGuard enforces
// nonce-uniqueness and a timestamp acceptance window on incoming requests.
package connections
