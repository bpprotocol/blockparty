import { hmacSha256, keccak256 } from "../crypto/index.js";
import { bytesToHex, utf8ToBytes } from "../bytes.js";
import type { World } from "./world.js";

/** Width of a type or audience code, in bytes. */
export const CODE_BYTES = 16;

/**
 * Normalization rule for derivation inputs: UTF-8 with no surrounding
 * whitespace and no trailing slashes.
 */
export function normalize(s: string): string {
  return s.trim().replace(/\/+$/, "");
}

/** Hex encoding of a code (or any byte string). */
export function codeHex(code: Uint8Array): string {
  return bytesToHex(code);
}

/** Derive the world-scoped 16-byte code for a canonical type URN. */
export function getTypeCode(world: World, canonicalType: string): Uint8Array {
  const seed = hmacSha256(world.typeSalt, utf8ToBytes(normalize(canonicalType)));
  return keccak256(seed).slice(0, CODE_BYTES);
}

/** Derive the world-scoped 16-byte code for an audience identifier. */
export function getAudienceCode(world: World, audienceID: string): Uint8Array {
  const seed = hmacSha256(world.audienceSalt, utf8ToBytes(normalize(audienceID)));
  return keccak256(seed).slice(0, CODE_BYTES);
}
