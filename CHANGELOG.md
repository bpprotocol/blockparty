# Changelog

All notable changes to the BlockParty Protocol will be documented in this file.

This project follows [Semantic Versioning](https://semver.org/).  
For a human-friendly overview, see the [whitepaper](./protocol/whitepaper.md) and [specs](./protocol/specs/).

---

## [0.1.0] — 2025-05-01

### Added
- Initial draft release of the BlockParty Protocol specification
- Core concepts: burnable identities, post-truth state, audience-scoped encryption
- Canonical block format spec (`block-format.md`)
- World and audience scoping system (`world-structure.md`)
- Identity and trust lifecycle design (`identity-burn-rfc.md`)
- Portable, offline-friendly transport model
- Protocol manifesto and founding essays

### Notes
- All documents released under [CC0 1.0 Public Domain Dedication](https://creativecommons.org/publicdomain/zero/1.0/)
- This version represents the philosophical and architectural foundation of the protocol

---

## [Unreleased]

### Changed — Whitepaper ↔ Spec alignment & crypto hardening
- Resolved contradictions between the whitepaper and specs by treating the specs as canonical:
  - Single canonical block-ID derivation (now binds the `data` payload) and identity address derivation (binds both PQ public keys).
  - `version` is a numeric `uint32` (matching `block.proto`), not a string.
  - Documented the distinct roles of the two chunking systems (`chunk.*` transport vs `content.chunked.*` content) and a shared reassembly algorithm.
- Hardened cryptography for post-quantum consistency:
  - Removed MD5 from type/audience code derivation in favor of Keccak-256.
  - Removed classical ECC from the core; World signing keys are now Dilithium, so `world_sig` is a genuine post-quantum signature.
  - Defined `GLOBAL_SALT` and the `DeterministicRNG` (SHAKE256) used for key generation.

### Added
- `specs/encryption.md` — audience-scoped AEAD (ML-KEM768 → HKDF-SHA256 → XChaCha20-Poly1305) with metadata bound as AAD.
- `specs/connections.md` — full `connect.*` post-quantum handshake, private-audience derivation, replay protection, rotation, and close.
- `specs/audiences.md` — audience model and the well-known public-audience derivation (`public-1`…`public-16`).
- Co-signature semantics (`sigs.extra[]`) in `block.md`.
- `identity.burn` block now reveals the root private keys, with verification and post-truth client behavior in the burn RFC.
- Proto fields for the connection handshake (`ConnectRequest`/`ConnectResponse`/`ConnectRotate`) and `IdentityBurn` revealed keys.

### Planned
- Extended commerce specification (`commerce.md`)
- RPC-over-block extensions for interactive applications
- Semantic content tagging and type resolution guide
- Protocol conformance recommendations for client implementers
