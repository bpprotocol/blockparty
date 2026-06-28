// Package crypto implements the BlockParty Protocol's cryptographic
// primitives and deterministic key generation.
//
// It is the foundation the rest of the SDK builds on (see issue #2). All
// algorithms follow the canonical definitions in protocol/specs/derivations.md:
//
//   - Hashing/KDF: Keccak-256, SHA-256, HMAC-SHA256, HKDF-SHA256.
//   - Deterministic RNG: a SHAKE256 XOF, used to drive reproducible key
//     generation.
//   - Post-quantum keys: ML-KEM768 (KEM) and Dilithium3 (signatures), via
//     Cloudflare CIRCL, derived deterministically from a seed.
//
// "Keccak-256" here is the original Keccak padding (as used by Ethereum), not
// FIPS-202 SHA3-256; the two produce different digests for the same input.
//
// Note on algorithm names: the protocol specifies "ML-KEM768" (FIPS 203) for
// the KEM and "Dilithium" for signatures. We therefore use CIRCL's FIPS ML-KEM
// for the KEM and round-3 Dilithium3 for signatures. Migrating signatures to
// the finalized ML-DSA (FIPS 204) is tracked as future work.
package crypto
