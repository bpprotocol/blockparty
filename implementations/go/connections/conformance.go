package connections

import "github.com/bpprotocol/blockparty/implementations/go/derive"

// The functions below expose the deterministic connection derivations (which are
// otherwise internal) for cross-client conformance vectors. The live handshake
// is non-deterministic (ephemeral keys, random nonces); these let other
// implementations check the pure derivations against shared fixtures.

// DeriveConnectionSecret0 is the epoch-0 connection secret from the two KEM
// shared secrets and the two handshake nonces.
func DeriveConnectionSecret0(ss1, ss2, nonce, nonceResp []byte) []byte {
	return connectionSecret0(ss1, ss2, nonce, nonceResp)
}

// DeriveRotatedSecret ratchets the connection secret to the given epoch with
// fresh KEM entropy ss3, salted by the rotation ciphertext ct3.
func DeriveRotatedSecret(prev, ss3, ct3 []byte, epoch uint64) []byte {
	return rotatedSecret(prev, ss3, ct3, epoch)
}

// DeriveAudienceSecret derives the private audience secret for an epoch.
func DeriveAudienceSecret(connSecret []byte, epoch uint64) []byte {
	return audienceSecret(connSecret, epoch)
}

// DeriveAudienceCode derives the private audience code for an epoch.
func DeriveAudienceCode(connSecret []byte, epoch uint64) derive.Code {
	return audienceCode(connSecret, epoch)
}
