// Package encryption implements BlockParty's audience-scoped AEAD layer
// (issue #5), following protocol/specs/encryption.md.
//
// Given a 32-byte audience secret, it encrypts and decrypts a block's data
// payload:
//
//   - ContentKey derives a per-block content key from the audience secret with
//     HKDF-SHA256, mixing in the block's nonce.
//   - EncryptData / DecryptData use XChaCha20-Poly1305, storing the 24-byte
//     nonce as a prefix of the data field and binding the block's visible
//     metadata (version, type_code, audience_code, timestamp) as additional
//     authenticated data (AAD), so tampering with that metadata fails
//     decryption.
//
// This package is agnostic to where the audience secret comes from: public
// audiences and connection (private) audiences derive it differently — see
// the audiences (#6) and connections (#7) packages. Blocks whose payload is
// inherently public (e.g. identity, identity.burn) skip this layer entirely
// and store their serialized payload directly in data.
package encryption
