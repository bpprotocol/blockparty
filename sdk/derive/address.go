package derive

import (
	"encoding/hex"

	"github.com/bpprotocol/blockparty/sdk/crypto"
)

// Address is a 40-character hex identity address.
type Address string

// BytesToAddress derives the canonical address binding both of an identity's
// post-quantum public keys: hex(Keccak256("v1" || mldsaPub || kyberPub)[-20:]).
func BytesToAddress(mldsaPub, kyberPub []byte) Address {
	digest := crypto.Keccak256([]byte("v1"), mldsaPub, kyberPub)
	return Address(hex.EncodeToString(digest[len(digest)-20:]))
}

// Bytes decodes the address into its 20 raw bytes.
func (a Address) Bytes() ([]byte, error) { return hex.DecodeString(string(a)) }
