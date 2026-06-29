import { create } from "@bufbuild/protobuf";
import { IdentityBurnSchema, type IdentityBurn } from "../blockpb/block_types_pb.js";
import { mldsaPublicFromSecret, kyberPublicFromSecret } from "../crypto/index.js";
import { bytesToAddress } from "../derive/index.js";
import type { Identity } from "../identity/index.js";

/**
 * Construct an identity.burn payload that reveals an identity's root private
 * keys, putting it into the post-truth state. notice is one of "voluntary",
 * "compromised", "rotated", or "other".
 */
export function buildBurn(id: Identity, notice: string): IdentityBurn {
  return create(IdentityBurnSchema, {
    identity: id.address,
    revealedMlDsa: id.mldsa.secretKey,
    revealedKyber: id.kyber.secretKey,
    burnNotice: notice,
  });
}

/**
 * Report whether a burn is genuine: the revealed private keys must re-derive
 * the public keys whose address matches burn.identity. No registry is needed —
 * the keys either reproduce the address or they do not.
 */
export function verifyBurn(burn: IdentityBurn): boolean {
  try {
    const mldsaPub = mldsaPublicFromSecret(burn.revealedMlDsa);
    const kyberPub = kyberPublicFromSecret(burn.revealedKyber);
    return bytesToAddress(mldsaPub, kyberPub) === burn.identity;
  } catch {
    return false; // malformed key bytes
  }
}

/**
 * Tracks which identities have been burned, so a client can flag their blocks
 * as contested. Not safe for concurrent mutation.
 */
export class TrustState {
  readonly #burned = new Set<string>();

  /**
   * Verify a burn and, if genuine, mark its identity burned. Returns whether the
   * burn was genuine; a burn whose keys do not match its claimed address is
   * ignored.
   */
  recordBurn(burn: IdentityBurn): boolean {
    if (!verifyBurn(burn)) return false;
    this.#burned.add(burn.identity);
    return true;
  }

  /**
   * Whether an identity address has been burned. Blocks authored by a burned
   * identity should be treated as contested.
   */
  isBurned(address: string): boolean {
    return this.#burned.has(address);
  }
}
