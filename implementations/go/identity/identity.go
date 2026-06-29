// Package identity implements BlockParty identity derivation (issue #3),
// following protocol/specs/identity.md.
//
// An identity is derived from a user passphrase scoped to a World: the World's
// WalletSalt namespaces the derivation so the same passphrase yields a
// different identity in each World. The signature and KEM keys use
// domain-separated seeds and are therefore independent.
package identity

import (
	"github.com/bpprotocol/blockparty/implementations/go/crypto"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
)

// info strings for the per-purpose HKDF derivations.
const (
	identityInfo = "bpprotocol.org/v1/identity"
	mldsaInfo    = "ml-dsa"
	mlkemInfo    = "mlkem"
)

// emptySalt is the empty (non-nil) HKDF salt the spec specifies for the
// per-key derivations (salt="").
var emptySalt = []byte{}

// Identity is a participant's post-quantum signing key, KEM key, and the
// address that binds them.
type Identity struct {
	Address derive.Address
	MLDSA   crypto.MLDSAKeyPair
	Kyber   crypto.KyberKeyPair
}

// OpenIdentity deterministically derives an identity from a passphrase within a
// World. Recovery is exact: the same (world, passphrase) always reproduces the
// same keys and address, and the same passphrase yields distinct identities
// across Worlds.
func OpenIdentity(w derive.World, passphrase string) Identity {
	worldPassword := crypto.HKDFSHA256([]byte(passphrase), w.WalletSalt, []byte(identityInfo), 32)

	mldsa := crypto.MakeMLDSAPair(crypto.HKDFSHA256(worldPassword, emptySalt, []byte(mldsaInfo), 32))
	kyber := crypto.MakeKyberPair(crypto.HKDFSHA256(worldPassword, emptySalt, []byte(mlkemInfo), 32))

	return Identity{
		Address: derive.BytesToAddress(mldsa.PublicBytes(), kyber.PublicBytes()),
		MLDSA:   mldsa,
		Kyber:   kyber,
	}
}
