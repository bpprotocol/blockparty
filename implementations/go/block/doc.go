// Package block implements the BlockParty block envelope (issue #4): canonical
// serialization, the two-layer signature model, and co-signatures, following
// protocol/specs/block.md.
//
// A block carries two required ML-DSA signatures and any number of optional
// co-signatures:
//
//   - world_sig  — by the World's signing key, over the block's fields.
//   - author_sig — by the author's signing key, over the fields plus world_sig.
//   - sigs.extra — co-signatures by other identities, each over the fields plus
//     world_sig and author_sig, verified independently.
//
// Signatures are computed over a domain-separated, length-prefixed preimage of
// the relevant fields (see preimage.go) rather than over the protobuf wire
// bytes, so signing is independent of encoder field ordering. The wire format
// itself is deterministic Protobuf (Encode/Decode).
//
// Typical flow: New → Sign → (AddCoSignature)* → Encode; and Decode → Verify.
package block
