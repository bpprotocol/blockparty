package audiences

import (
	"strconv"

	"github.com/bpprotocol/blockparty/implementations/go/crypto"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
)

const (
	// PublicAudiencePrefix is the namespace for the well-known public audiences.
	PublicAudiencePrefix = "bpprotocol.org/v1/audience/public-"
	// InboxAudiencePrefix is the namespace for per-identity inbox audiences.
	InboxAudiencePrefix = "bpprotocol.org/v1/audience/inbox/"
	// ReservedPublicCount is the number of reserved well-known public audiences
	// (public-1 .. public-16). Higher numbers are valid but interop is only
	// guaranteed for the reserved set.
	ReservedPublicCount = 16

	// audienceSecretInfo is the HKDF info for the public-audience secret.
	audienceSecretInfo = "bpprotocol.org/v1/audience-secret"
)

// Audience is a derived audience: its identifier, its world-scoped code
// (addressing label), and — for confidential audiences — its 32-byte secret.
// Rendezvous-only audiences (e.g. inbox) have a nil Secret.
type Audience struct {
	ID     string
	Code   derive.Code
	Secret []byte
}

// PublicAudienceID returns the canonical identifier for public audience n.
func PublicAudienceID(n int) string {
	return PublicAudiencePrefix + strconv.Itoa(n)
}

// IsReservedPublic reports whether n is in the reserved well-known range.
func IsReservedPublic(n int) bool { return n >= 1 && n <= ReservedPublicCount }

// PublicAudienceSecret derives the 32-byte secret for a public audience within
// a World. Per encryption.md it is HKDF-SHA256 over the World's AudienceSalt,
// salted by the audience code (the hex form, as returned by GetAudienceCode),
// with a fixed info. Only holders of the World seed can derive it.
func PublicAudienceSecret(w derive.World, audienceID string) []byte {
	code := derive.GetAudienceCode(w, audienceID)
	return crypto.HKDFSHA256(w.AudienceSalt, []byte(code.Hex()), []byte(audienceSecretInfo), 32)
}

// PublicAudience derives the full public audience (code + secret) for public-n
// within a World.
func PublicAudience(w derive.World, n int) Audience {
	id := PublicAudienceID(n)
	return Audience{
		ID:     id,
		Code:   derive.GetAudienceCode(w, id),
		Secret: PublicAudienceSecret(w, id),
	}
}

// ReservedPublicAudiences derives all reserved public audiences (public-1 ..
// public-16) for a World.
func ReservedPublicAudiences(w derive.World) []Audience {
	out := make([]Audience, 0, ReservedPublicCount)
	for n := 1; n <= ReservedPublicCount; n++ {
		out = append(out, PublicAudience(w, n))
	}
	return out
}

// InboxAudienceID returns the rendezvous audience identifier for an address.
func InboxAudienceID(addr derive.Address) string {
	return InboxAudiencePrefix + string(addr)
}

// InboxAudience derives the rendezvous audience for delivering connection
// handshake blocks to an address. It has a code but no secret: the inbox is a
// rendezvous, and handshake confidentiality comes from the KEM, not the inbox.
func InboxAudience(w derive.World, addr derive.Address) Audience {
	id := InboxAudienceID(addr)
	return Audience{ID: id, Code: derive.GetAudienceCode(w, id)}
}
