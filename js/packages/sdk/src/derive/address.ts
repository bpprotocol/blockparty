import { keccak256 } from "../crypto/index.js";
import { bytesToHex, hexToBytes, utf8ToBytes } from "../bytes.js";

/**
 * Derive the canonical 40-char hex address binding both of an identity's
 * post-quantum public keys: hex(keccak256("v1" || mldsaPub || kyberPub)[-20:]).
 */
export function bytesToAddress(mldsaPub: Uint8Array, kyberPub: Uint8Array): string {
  const digest = keccak256(utf8ToBytes("v1"), mldsaPub, kyberPub);
  return bytesToHex(digest.subarray(digest.length - 20));
}

/** Decode an address into its 20 raw bytes. */
export function addressBytes(address: string): Uint8Array {
  return hexToBytes(address);
}
