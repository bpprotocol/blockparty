package node

import (
	"bytes"

	"github.com/bpprotocol/blockparty/sdk/block"
	"github.com/bpprotocol/blockparty/sdk/blockpb"
	"github.com/bpprotocol/blockparty/sdk/blocktypes"
	"github.com/bpprotocol/blockparty/sdk/derive"
	"github.com/bpprotocol/blockparty/sdk/encryption"
	"github.com/bpprotocol/blockparty/sdk/identity"
)

// ContentTypeHeader is the required MIME header key for content payloads.
const ContentTypeHeader = "bpprotocol.org/v1/content-type"

// BuildPost creates a signed content.post block whose text body is encrypted to
// an audience (identified by audienceCode + audienceSecret) within a World.
func BuildPost(w derive.World, author identity.Identity, audienceCode derive.Code, audienceSecret []byte, timestamp int64, text string) (*blockpb.Block, error) {
	typeCode := derive.GetTypeCode(w, blocktypes.TypeContentPost)
	payload, err := blocktypes.MarshalPayload(&blockpb.ContentPost{
		Headers: map[string]string{ContentTypeHeader: "text/plain"},
		Body:    []byte(text),
	})
	if err != nil {
		return nil, err
	}
	data, err := encryption.EncryptForBlock(audienceSecret, block.Version, []byte(typeCode), []byte(audienceCode), timestamp, payload)
	if err != nil {
		return nil, err
	}
	b := block.New(author.Address, typeCode, audienceCode, timestamp, data)
	block.Sign(b, w, author.MLDSA)
	return b, nil
}

// OpenPost decrypts and decodes a content.post block with a known audience
// secret, returning the text body. ok is false if the block is not a
// content.post for this World. Decryption is authenticated (AEAD), so a
// tampered block yields an error.
func OpenPost(w derive.World, r *blocktypes.Resolver, b *blockpb.Block, audienceSecret []byte, text *string) (ok bool, err error) {
	urn, found := r.TypeURN(b.TypeCode)
	if !found || urn != blocktypes.TypeContentPost {
		return false, nil
	}
	plaintext, err := encryption.DecryptData(audienceSecret, b)
	if err != nil {
		return false, err
	}
	msg, err := blocktypes.DecodePayload(urn, plaintext)
	if err != nil {
		return false, err
	}
	post, ok := msg.(*blockpb.ContentPost)
	if !ok {
		return false, nil
	}
	*text = string(bytes.Clone(post.Body))
	return true, nil
}
