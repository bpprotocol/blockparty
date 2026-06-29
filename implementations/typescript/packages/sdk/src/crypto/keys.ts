import { ml_kem768 } from "@noble/post-quantum/ml-kem.js";
import { ml_dsa65 } from "@noble/post-quantum/ml-dsa.js";
import { keccak256 } from "./hash.js";
import { deterministicRNG } from "./rng.js";

/** Seed sizes consumed by the standardized key-generation routines. */
export const KYBER_SEED_SIZE = 64; // ML-KEM768
export const MLDSA_SEED_SIZE = 32; // ML-DSA-65

/** A post-quantum keypair (raw bytes, as produced by @noble/post-quantum). */
export interface KeyPair {
  readonly publicKey: Uint8Array;
  readonly secretKey: Uint8Array;
}

/** The output of an ML-KEM768 encapsulation. */
export interface Encapsulation {
  readonly ciphertext: Uint8Array;
  readonly sharedSecret: Uint8Array;
}

/**
 * Deterministically derive an ML-KEM768 keypair from a seed: the seed is reduced
 * with keccak256, expanded through a SHAKE256 XOF, and the scheme's seed bytes
 * drive the standardized FIPS-203 key generation. The same seed always yields
 * the same keypair, byte-identical to the Go reference.
 */
export function makeKyberPair(seed: Uint8Array): KeyPair {
  return ml_kem768.keygen(deterministicRNG(keccak256(seed), KYBER_SEED_SIZE));
}

/** Deterministically derive an ML-DSA-65 (FIPS 204) keypair from a seed. */
export function makeMldsaPair(seed: Uint8Array): KeyPair {
  return ml_dsa65.keygen(deterministicRNG(keccak256(seed), MLDSA_SEED_SIZE));
}

/**
 * Produce an ML-DSA-65 signature over message. Signing is **deterministic**
 * (`extraEntropy: false`) so signatures are reproducible and match the Go
 * reference.
 */
export function signMldsa(secretKey: Uint8Array, message: Uint8Array): Uint8Array {
  return ml_dsa65.sign(message, secretKey, { extraEntropy: false });
}

/** Verify an ML-DSA-65 signature. */
export function verifyMldsa(
  publicKey: Uint8Array,
  message: Uint8Array,
  signature: Uint8Array,
): boolean {
  return ml_dsa65.verify(signature, message, publicKey);
}

/** Generate a fresh shared secret for a public key and encapsulate it. */
export function encapsulate(publicKey: Uint8Array): Encapsulation {
  const { cipherText, sharedSecret } = ml_kem768.encapsulate(publicKey);
  return { ciphertext: cipherText, sharedSecret };
}

/** Recover the shared secret encapsulated in ciphertext for a secret key. */
export function decapsulate(ciphertext: Uint8Array, secretKey: Uint8Array): Uint8Array {
  return ml_kem768.decapsulate(ciphertext, secretKey);
}

/** Recover the ML-DSA-65 public key from its secret key (e.g. to verify a burn). */
export function mldsaPublicFromSecret(secretKey: Uint8Array): Uint8Array {
  return ml_dsa65.getPublicKey(secretKey);
}

/** Recover the ML-KEM768 public key from its secret key. */
export function kyberPublicFromSecret(secretKey: Uint8Array): Uint8Array {
  return ml_kem768.getPublicKey(secretKey);
}
