import { hkdfSha256, makeKyberPair, makeMldsaPair, type KeyPair } from "../crypto/index.js";
import { bytesToAddress, type World } from "../derive/index.js";
import { utf8ToBytes } from "../bytes.js";

// HKDF info strings for the per-purpose derivations.
const IDENTITY_INFO = "bpprotocol.org/v1/identity";
const MLDSA_INFO = "ml-dsa";
const MLKEM_INFO = "mlkem";

// The empty (non-nil) HKDF salt the spec specifies for the per-key derivations.
const EMPTY_SALT = new Uint8Array(0);

/** A participant's ML-DSA-65 signing key, ML-KEM768 key, and bound address. */
export interface Identity {
  readonly address: string;
  readonly mldsa: KeyPair;
  readonly kyber: KeyPair;
}

/**
 * Deterministically derive an identity from a passphrase within a World. The
 * World's wallet salt namespaces the derivation, so the same passphrase yields
 * a different identity per World; the signature and KEM keys use
 * domain-separated seeds and are independent. Recovery is exact.
 */
export function openIdentity(world: World, passphrase: string): Identity {
  const worldPassword = hkdfSha256(
    utf8ToBytes(passphrase),
    world.walletSalt,
    utf8ToBytes(IDENTITY_INFO),
    32,
  );
  const mldsa = makeMldsaPair(hkdfSha256(worldPassword, EMPTY_SALT, utf8ToBytes(MLDSA_INFO), 32));
  const kyber = makeKyberPair(hkdfSha256(worldPassword, EMPTY_SALT, utf8ToBytes(MLKEM_INFO), 32));
  return { address: bytesToAddress(mldsa.publicKey, kyber.publicKey), mldsa, kyber };
}
