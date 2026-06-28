package derive

import (
	"encoding/hex"

	"github.com/bpprotocol/blockparty/sdk/crypto"
)

// Address is a 40-character hex identity address.
type Address string

// BytesToAddress derives the canonical address binding both of an identity's
// post-quantum public keys: hex(Keccak256("v1" || dilithiumPub || kyberPub)[-20:]).
func BytesToAddress(dilithiumPub, kyberPub []byte) Address {
	digest := crypto.Keccak256([]byte("v1"), dilithiumPub, kyberPub)
	return Address(hex.EncodeToString(digest[len(digest)-20:]))
}

// Bytes decodes the address into its 20 raw bytes.
func (a Address) Bytes() ([]byte, error) { return hex.DecodeString(string(a)) }
