// Package blocktypes implements the core BlockParty block types and their
// handlers (issue #8), following protocol/specs/block-types.md and rpc.md.
//
// The wire payload structs are generated in blockpb; this package adds:
//
//   - Canonical type URNs (TypeContentPost, ...) and a factory/Resolver that
//     maps a block's world-scoped type_code back to its URN and decodes the
//     payload.
//   - The rpc.render guard: a render block is local-only, so decoding one that
//     arrived over a transport is refused (ErrRenderNotTransportable).
//   - Chunk reassembly for both chunk.* and content.chunked.*: concatenate the
//     referenced chunk payloads in manifest order and verify the SHA-512.
//   - identity.burn verification (the revealed private keys must re-derive the
//     burned address) and a TrustState that flags a burned identity's blocks
//     as contested.
//   - The rpc/core.ping request/response helpers.
package blocktypes
