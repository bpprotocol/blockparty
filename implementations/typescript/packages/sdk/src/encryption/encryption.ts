import { xchacha20poly1305 } from "@noble/ciphers/chacha.js";
import { randomBytes } from "@noble/hashes/utils.js";
import { hkdfSha256 } from "../crypto/index.js";
import { concatBytes, utf8ToBytes } from "../bytes.js";
import type { Block } from "../blockpb/block_pb.js";

/** XChaCha20-Poly1305 nonce length, prefixed to the data field. */
export const NONCE_SIZE = 24;

// HKDF info prefix for content-key derivation; the block nonce is appended.
const AEAD_INFO = "bpprotocol.org/v1/aead";

/**
 * Derive the 32-byte per-block content key from the audience secret, audience
 * code, and block nonce (see protocol/specs/encryption.md).
 */
export function contentKey(
  audienceSecret: Uint8Array,
  audienceCode: Uint8Array,
  blockNonce: Uint8Array,
): Uint8Array {
  const info = concatBytes(utf8ToBytes(AEAD_INFO), blockNonce);
  return hkdfSha256(audienceSecret, audienceCode, info, 32);
}

/**
 * Build the additional authenticated data bound to a block's ciphertext: the
 * 4-byte big-endian version, the raw type and audience codes, then the 8-byte
 * big-endian timestamp.
 */
export function aad(
  version: number,
  typeCode: Uint8Array,
  audienceCode: Uint8Array,
  timestamp: number | bigint,
): Uint8Array {
  const head = new Uint8Array(4);
  new DataView(head.buffer).setUint32(0, version >>> 0, false);
  const tail = new Uint8Array(8);
  new DataView(tail.buffer).setBigUint64(0, BigInt.asUintN(64, BigInt(timestamp)), false);
  return concatBytes(head, typeCode, audienceCode, tail);
}

/**
 * Deterministic AEAD seal with a caller-supplied nonce: returns
 * `nonce || ciphertext+tag`. Exposed for conformance vectors and deterministic
 * testing; normal callers use encryptData / encryptForBlock.
 */
export function sealWithNonce(
  audienceSecret: Uint8Array,
  audienceCode: Uint8Array,
  nonce: Uint8Array,
  plaintext: Uint8Array,
  aadBytes: Uint8Array,
): Uint8Array {
  const key = contentKey(audienceSecret, audienceCode, nonce);
  const ciphertext = xchacha20poly1305(key, nonce, aadBytes).encrypt(plaintext);
  return concatBytes(nonce, ciphertext);
}

/**
 * Encrypt plaintext for an audience and return the data field: a fresh 24-byte
 * nonce followed by the AEAD ciphertext (which includes the Poly1305 tag).
 */
export function encryptData(
  audienceSecret: Uint8Array,
  audienceCode: Uint8Array,
  plaintext: Uint8Array,
  aadBytes: Uint8Array,
): Uint8Array {
  return sealWithNonce(audienceSecret, audienceCode, randomBytes(NONCE_SIZE), plaintext, aadBytes);
}

/**
 * Convenience that builds the AAD from a block's metadata and encrypts plaintext
 * into a data field. The caller then constructs the block with the returned data
 * (so the block ID binds the ciphertext).
 */
export function encryptForBlock(
  audienceSecret: Uint8Array,
  version: number,
  typeCode: Uint8Array,
  audienceCode: Uint8Array,
  timestamp: number | bigint,
  plaintext: Uint8Array,
): Uint8Array {
  return encryptData(
    audienceSecret,
    audienceCode,
    plaintext,
    aad(version, typeCode, audienceCode, timestamp),
  );
}

/**
 * Decrypt a block's data field using the audience secret. The AAD is
 * reconstructed from the block's metadata, so tampering with version, type_code,
 * audience_code, or timestamp causes authentication to fail (decrypt throws).
 * A caller who cannot derive the audience secret cannot decrypt.
 */
export function decryptData(audienceSecret: Uint8Array, block: Block): Uint8Array {
  if (block.data.length < NONCE_SIZE) {
    throw new Error("encryption: data shorter than nonce");
  }
  const nonce = block.data.subarray(0, NONCE_SIZE);
  const ciphertext = block.data.subarray(NONCE_SIZE);
  const key = contentKey(audienceSecret, block.audienceCode, nonce);
  const aadBytes = aad(block.version, block.typeCode, block.audienceCode, block.timestamp);
  return xchacha20poly1305(key, nonce, aadBytes).decrypt(ciphertext);
}
