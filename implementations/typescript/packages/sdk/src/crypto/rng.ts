import { shake256 } from "@noble/hashes/sha3.js";

/**
 * Deterministic byte stream: the first `length` bytes of a SHAKE256 XOF that
 * absorbs `seed`. The same seed always yields the same bytes, which makes key
 * generation reproducible across implementations.
 *
 * Per protocol/specs/derivations.md, key-derivation callers first reduce their
 * input to 32 bytes with keccak256 and pass that as `seed` (see makeKyberPair /
 * makeMldsaPair).
 */
export function deterministicRNG(seed: Uint8Array, length: number): Uint8Array {
  return shake256(seed, { dkLen: length });
}
