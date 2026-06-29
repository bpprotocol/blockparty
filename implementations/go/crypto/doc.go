// Package crypto implements the BlockParty Protocol's cryptographic
// primitives and deterministic key generation.
//
// It is the foundation the rest of the SDK builds on (see issue #2). All
// algorithms follow the canonical definitions in protocol/specs/derivations.md:
//
//   - Hashing/KDF: Keccak-256, SHA-256, HMAC-SHA256, HKDF-SHA256.
//   - Deterministic RNG: a SHAKE256 XOF, used to drive reproducible key
//     generation.
//   - Post-quantum keys: ML-KEM768 (KEM) and ML-DSA-65 (signatures), via
//     Cloudflare CIRCL, derived deterministically from a seed.
//
// "Keccak-256" here is the original Keccak padding (as used by Ethereum), not
// FIPS-202 SHA3-256; the two produce different digests for the same input.
//
// The protocol uses FIPS-203 ML-KEM768 for the KEM and FIPS-204 ML-DSA-65 for
// signatures, both via Cloudflare CIRCL.
package crypto
