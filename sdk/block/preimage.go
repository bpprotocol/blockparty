package block

import (
	"encoding/binary"

	"github.com/bpprotocol/blockparty/sdk/blockpb"
)

// Domain separators for the three signature layers. Including a distinct domain
// in each preimage prevents a signature from one layer being replayed as
// another.
const (
	worldDomain  = "bpprotocol.org/v1/sig/world"
	authorDomain = "bpprotocol.org/v1/sig/author"
	cosignDomain = "bpprotocol.org/v1/sig/cosign"
)

// preimage builds a deterministic signing preimage: the domain tag followed by
// each field, every element framed with an 8-byte big-endian length prefix so
// no concatenation of fields is ambiguous.
func preimage(domain string, fields ...[]byte) []byte {
	out := appendField(nil, []byte(domain))
	for _, f := range fields {
		out = appendField(out, f)
	}
	return out
}

func appendField(buf, f []byte) []byte {
	var l [8]byte
	binary.BigEndian.PutUint64(l[:], uint64(len(f)))
	buf = append(buf, l[:]...)
	return append(buf, f...)
}

func u32(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

func i64(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

// metaFields are the core fields every signature layer commits to.
func metaFields(b *blockpb.Block) [][]byte {
	return [][]byte{u32(b.Version), b.Id, b.TypeCode, b.AudienceCode, i64(b.Timestamp), b.Data}
}

// worldPreimage is signed by the World key: the core fields.
func worldPreimage(b *blockpb.Block) []byte {
	return preimage(worldDomain, metaFields(b)...)
}

// authorPreimage is signed by the author: the core fields plus world_sig.
func authorPreimage(b *blockpb.Block) []byte {
	return preimage(authorDomain, append(metaFields(b), worldSig(b))...)
}

// coSignPreimage is signed by a co-signer: the core fields plus world_sig and
// author_sig. It does not include other co-signatures, so co-signatures are
// order-independent and verified independently of one another.
func coSignPreimage(b *blockpb.Block) []byte {
	return preimage(cosignDomain, append(metaFields(b), worldSig(b), authorSig(b))...)
}

func worldSig(b *blockpb.Block) []byte {
	if b.Sigs == nil {
		return nil
	}
	return b.Sigs.World
}

func authorSig(b *blockpb.Block) []byte {
	if b.Sigs == nil {
		return nil
	}
	return b.Sigs.Author
}
