// Package derive implements the deterministic identifier and key derivations
// defined in protocol/specs/derivations.md (issue #3).
//
// Everything here is reproducible from inputs alone, with no central registry:
//
//   - Worlds: OpenWorld / GenerateWorld produce a World's ML-DSA signing key
//     and its wallet/type/audience salts from a seed phrase.
//   - Codes: GetTypeCode / GetAudienceCode hash an identifier under a World salt
//     into a 16-byte, world-scoped code.
//   - Addresses: BytesToAddress binds an identity's two public keys into a
//     40-char hex address.
//   - Block IDs: GetBlockID derives a content-binding identifier for a block.
//
// String inputs are normalized (trimmed of surrounding whitespace and trailing
// slashes) per the spec's normalization rule before hashing.
package derive
