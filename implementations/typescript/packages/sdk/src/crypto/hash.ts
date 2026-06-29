import { keccak_256 } from "@noble/hashes/sha3.js";
import { sha256 as _sha256, sha512 as _sha512 } from "@noble/hashes/sha2.js";
import { hmac } from "@noble/hashes/hmac.js";
import { hkdf } from "@noble/hashes/hkdf.js";
import { concatBytes } from "@noble/hashes/utils.js";

/**
 * Keccak-256 (original Keccak padding, as used by Ethereum) of the concatenated
 * parts. Distinct from FIPS-202 SHA3-256.
 */
export function keccak256(...parts: Uint8Array[]): Uint8Array {
  return keccak_256(concatBytes(...parts));
}

/** SHA-256 of the concatenated parts. */
export function sha256(...parts: Uint8Array[]): Uint8Array {
  return _sha256(concatBytes(...parts));
}

/** SHA-512 of the concatenated parts. */
export function sha512(...parts: Uint8Array[]): Uint8Array {
  return _sha512(concatBytes(...parts));
}

/** HMAC-SHA256 of message under key. */
export function hmacSha256(key: Uint8Array, message: Uint8Array): Uint8Array {
  return hmac(_sha256, key, message);
}

/**
 * HKDF (extract-and-expand) with SHA-256, producing `length` bytes. `salt` and
 * `info` are passed verbatim; an empty `salt` is used as an empty HMAC key
 * (matching the Go reference), not zero-filled.
 */
export function hkdfSha256(
  ikm: Uint8Array,
  salt: Uint8Array,
  info: Uint8Array,
  length: number,
): Uint8Array {
  return hkdf(_sha256, ikm, salt, info, length);
}
