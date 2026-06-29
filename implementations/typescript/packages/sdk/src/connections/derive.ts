import { hkdfSha256, keccak256 } from "../crypto/index.js";
import { CODE_BYTES } from "../derive/index.js";
import { concatBytes, utf8ToBytes } from "../bytes.js";

const connectInfo = (epoch: number): string => `bpprotocol.org/v1/connect:epoch=${epoch}`;
const audienceInfo = (epoch: number): string => `bpprotocol.org/v1/connect-audience:epoch=${epoch}`;

const EMPTY_SALT = new Uint8Array(0);

/**
 * The epoch-0 connection secret from the two KEM shared secrets and the two
 * handshake nonces. Both parties hold ss1, ss2, nonce, and nonceResp, so both
 * derive the same secret.
 */
export function connectionSecret0(
  ss1: Uint8Array,
  ss2: Uint8Array,
  nonce: Uint8Array,
  nonceResp: Uint8Array,
): Uint8Array {
  return hkdfSha256(
    concatBytes(ss1, ss2),
    concatBytes(nonce, nonceResp),
    utf8ToBytes(connectInfo(0)),
    32,
  );
}

/**
 * Ratchet the connection secret to the next epoch with fresh KEM entropy ss3,
 * salted by the rotation ciphertext ct3.
 */
export function rotatedSecret(
  prev: Uint8Array,
  ss3: Uint8Array,
  ct3: Uint8Array,
  epoch: number,
): Uint8Array {
  return hkdfSha256(concatBytes(prev, ss3), ct3, utf8ToBytes(connectInfo(epoch)), 32);
}

/** The private audience secret for an epoch (fed to the AEAD layer). */
export function audienceSecret(connSecret: Uint8Array, epoch: number): Uint8Array {
  return hkdfSha256(connSecret, EMPTY_SALT, utf8ToBytes(audienceInfo(epoch)), 32);
}

/**
 * The private audience code for an epoch. Derived from the secret, so it is
 * unlinkable to either participant's address.
 */
export function audienceCode(connSecret: Uint8Array, epoch: number): Uint8Array {
  return keccak256(audienceSecret(connSecret, epoch)).slice(0, CODE_BYTES);
}
