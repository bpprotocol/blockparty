package derive

import (
	"encoding/hex"
	"strings"

	"github.com/bpprotocol/blockparty/sdk/crypto"
)

// CodeBytes is the width of a type or audience code.
const CodeBytes = 16

// Code is a 16-byte, world-scoped identifier (a type code or audience code).
// The raw bytes are stored in a block's metadata; Hex is used where the spec's
// derivations operate on the textual form (e.g. GetBlockID).
type Code []byte

// Hex returns the lowercase hex encoding of the code.
func (c Code) Hex() string { return hex.EncodeToString(c) }

// GetTypeCode derives the world-scoped code for a canonical type URN, e.g.
// "bpprotocol.org/v1/types/content.post".
func GetTypeCode(w World, canonicalType string) Code {
	seed := crypto.HMACSHA256(w.TypeSalt, []byte(normalize(canonicalType)))
	return Code(crypto.Keccak256(seed)[:CodeBytes])
}

// GetAudienceCode derives the world-scoped code for an audience identifier,
// e.g. "bpprotocol.org/v1/audience/public-1".
func GetAudienceCode(w World, audienceID string) Code {
	seed := crypto.HMACSHA256(w.AudienceSalt, []byte(normalize(audienceID)))
	return Code(crypto.Keccak256(seed)[:CodeBytes])
}

// normalize applies the spec's normalization rule: UTF-8 input with no
// surrounding whitespace and no trailing slashes.
func normalize(s string) string {
	return strings.TrimRight(strings.TrimSpace(s), "/")
}
