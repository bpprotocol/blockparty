import { hmacSha256, makeMldsaPair, type KeyPair } from "../crypto/index.js";
import { utf8ToBytes } from "../bytes.js";

/**
 * Fixed, non-secret domain-separation salt used to open a World from a seed
 * phrase. Hard-coded into every conforming client.
 */
export const GLOBAL_SALT = "bpprotocol.org/v1/global";

/**
 * A cryptographic domain: an ML-DSA-65 signing key (the world_sig authority)
 * plus the three salts that scope identity, type, and audience derivations.
 */
export interface World {
  readonly signingKey: KeyPair;
  readonly walletSalt: Uint8Array;
  readonly typeSalt: Uint8Array;
  readonly audienceSalt: Uint8Array;
}

/**
 * Deterministically derive a World from a seed phrase. Anyone who knows the
 * phrase derives the identical World, including its signing key.
 */
export function openWorld(seedPhrase: string): World {
  const worldSeed = hmacSha256(utf8ToBytes(GLOBAL_SALT), utf8ToBytes(seedPhrase));
  return generateWorld(makeMldsaPair(worldSeed), worldSeed);
}

/** Build a World from a signing key and the world seed (mirrors the spec). */
export function generateWorld(signingKey: KeyPair, worldSeed: Uint8Array): World {
  return {
    signingKey,
    walletSalt: hmacSha256(worldSeed, utf8ToBytes("wallets")),
    typeSalt: hmacSha256(worldSeed, utf8ToBytes("types")),
    audienceSalt: hmacSha256(worldSeed, utf8ToBytes("channels")),
  };
}
