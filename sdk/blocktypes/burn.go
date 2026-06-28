package blocktypes

import (
	"errors"

	"github.com/cloudflare/circl/sign"

	"github.com/bpprotocol/blockparty/sdk/blockpb"
	"github.com/bpprotocol/blockparty/sdk/crypto"
	"github.com/bpprotocol/blockparty/sdk/derive"
	"github.com/bpprotocol/blockparty/sdk/identity"
)

// BuildBurn constructs an identity.burn payload that reveals an identity's root
// private keys, putting it into the post-truth state. notice is one of
// "voluntary", "compromised", "rotated", or "other".
func BuildBurn(id identity.Identity, notice string) (*blockpb.IdentityBurn, error) {
	dil, err := id.Dilithium.Private.MarshalBinary()
	if err != nil {
		return nil, err
	}
	kyb, err := id.Kyber.Private.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &blockpb.IdentityBurn{
		Identity:          string(id.Address),
		RevealedDilithium: dil,
		RevealedKyber:     kyb,
		BurnNotice:        notice,
	}, nil
}

// VerifyBurn reports whether a burn is genuine: the revealed private keys must
// re-derive the public keys whose address matches burn.Identity. No registry is
// needed — the keys either reproduce the address or they do not.
func VerifyBurn(burn *blockpb.IdentityBurn) (bool, error) {
	sk, err := crypto.SigScheme().UnmarshalBinaryPrivateKey(burn.RevealedDilithium)
	if err != nil {
		return false, err
	}
	dilPub, ok := sk.Public().(sign.PublicKey)
	if !ok {
		return false, errors.New("blocktypes: revealed dilithium key has no public key")
	}
	kk, err := crypto.KEMScheme().UnmarshalBinaryPrivateKey(burn.RevealedKyber)
	if err != nil {
		return false, err
	}
	dilBytes, err := dilPub.MarshalBinary()
	if err != nil {
		return false, err
	}
	kybBytes, err := kk.Public().MarshalBinary()
	if err != nil {
		return false, err
	}
	return string(derive.BytesToAddress(dilBytes, kybBytes)) == burn.Identity, nil
}

// TrustState tracks which identities have been burned, so a client can flag
// their blocks as contested. It is not safe for concurrent mutation.
type TrustState struct {
	burned map[string]struct{}
}

// NewTrustState returns an empty trust state.
func NewTrustState() *TrustState {
	return &TrustState{burned: make(map[string]struct{})}
}

// RecordBurn verifies a burn and, if genuine, marks its identity burned. It
// returns whether the burn was genuine. A burn whose keys do not match its
// claimed address is ignored.
func (t *TrustState) RecordBurn(burn *blockpb.IdentityBurn) (bool, error) {
	ok, err := VerifyBurn(burn)
	if err != nil || !ok {
		return false, err
	}
	t.burned[burn.Identity] = struct{}{}
	return true, nil
}

// IsBurned reports whether an identity address has been burned. Blocks authored
// by a burned identity should be treated as contested (pre- and post-burn are
// indistinguishable once the keys are public).
func (t *TrustState) IsBurned(address string) bool {
	_, ok := t.burned[address]
	return ok
}
