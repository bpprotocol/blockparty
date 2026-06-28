package block

import (
	"encoding/hex"
	"errors"

	"github.com/cloudflare/circl/sign"
	"google.golang.org/protobuf/proto"

	"github.com/bpprotocol/blockparty/sdk/blockpb"
	"github.com/bpprotocol/blockparty/sdk/crypto"
	"github.com/bpprotocol/blockparty/sdk/derive"
)

// Version is the numeric protocol version stamped on new blocks.
const Version uint32 = 1

var (
	// ErrMissingSignatures is returned when a required signature is absent.
	ErrMissingSignatures = errors.New("block: missing required signature")
	// ErrWorldSignature is returned when the world signature does not verify.
	ErrWorldSignature = errors.New("block: invalid world signature")
	// ErrAuthorSignature is returned when the author signature does not verify.
	ErrAuthorSignature = errors.New("block: invalid author signature")
)

// New constructs an unsigned block with its content-binding ID derived from the
// author's address (see derive.GetBlockID). Call Sign next.
func New(authorAddress derive.Address, typeCode, audienceCode derive.Code, timestamp int64, data []byte) *blockpb.Block {
	idHex := derive.GetBlockID(Version, timestamp, audienceCode, authorAddress, typeCode, data)
	id, err := hex.DecodeString(idHex)
	if err != nil {
		// GetBlockID always returns valid hex.
		panic("block: derived id is not valid hex: " + err.Error())
	}
	return &blockpb.Block{
		Version:      Version,
		Id:           id,
		TypeCode:     []byte(typeCode),
		AudienceCode: []byte(audienceCode),
		Timestamp:    timestamp,
		Data:         data,
	}
}

// IDHex returns the block's ID as a lowercase hex string.
func IDHex(b *blockpb.Block) string { return hex.EncodeToString(b.Id) }

// Sign applies the world signature (with the World's signing key) and then the
// author signature (with the author's signing key). It overwrites any existing
// world/author signatures and leaves co-signatures untouched.
func Sign(b *blockpb.Block, world derive.World, author crypto.DilithiumKeyPair) {
	if b.Sigs == nil {
		b.Sigs = &blockpb.Signatures{}
	}
	b.Sigs.World = world.SigningKey.Sign(worldPreimage(b))
	b.Sigs.Author = author.Sign(authorPreimage(b))
}

// Verify checks the two required signatures: world_sig against the World's
// signing key and author_sig against authorPub. It ignores co-signatures, so a
// missing or invalid co-signature never invalidates a block. Tampering with any
// signed field causes one of the checks to fail.
func Verify(b *blockpb.Block, world derive.World, authorPub sign.PublicKey) error {
	if b.Sigs == nil || len(b.Sigs.World) == 0 || len(b.Sigs.Author) == 0 {
		return ErrMissingSignatures
	}
	if !crypto.VerifyDilithium(world.SigningKey.Public, worldPreimage(b), b.Sigs.World) {
		return ErrWorldSignature
	}
	if !crypto.VerifyDilithium(authorPub, authorPreimage(b), b.Sigs.Author) {
		return ErrAuthorSignature
	}
	return nil
}

// AddCoSignature appends a co-signature of the given namespaced type, signed by
// signer over the block's fields plus its world and author signatures. The
// block must already be signed (Sign) so author_sig is present.
func AddCoSignature(b *blockpb.Block, sigType string, signer crypto.DilithiumKeyPair) {
	if b.Sigs == nil {
		b.Sigs = &blockpb.Signatures{}
	}
	b.Sigs.Extra = append(b.Sigs.Extra, &blockpb.ExtraSignature{
		Type: sigType,
		Sig:  signer.Sign(coSignPreimage(b)),
	})
}

// VerifyCoSignature reports whether extra is a valid co-signature of b by
// signerPub.
func VerifyCoSignature(b *blockpb.Block, extra *blockpb.ExtraSignature, signerPub sign.PublicKey) bool {
	return crypto.VerifyDilithium(signerPub, coSignPreimage(b), extra.Sig)
}

// CoSignatures returns the block's co-signatures (sigs.extra), or nil.
func CoSignatures(b *blockpb.Block) []*blockpb.ExtraSignature {
	if b.Sigs == nil {
		return nil
	}
	return b.Sigs.Extra
}

// Encode serializes a block to its canonical, deterministic Protobuf wire form.
func Encode(b *blockpb.Block) ([]byte, error) {
	return proto.MarshalOptions{Deterministic: true}.Marshal(b)
}

// Decode parses a block from its Protobuf wire form.
func Decode(data []byte) (*blockpb.Block, error) {
	var b blockpb.Block
	if err := proto.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	return &b, nil
}
