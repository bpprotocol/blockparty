import { bytesToHex } from "../bytes.js";
import type { Audience } from "./audiences.js";

/**
 * A local map from audience code to the audience a client knows, letting it
 * find the secret for a received block's audience_code. This is the "local
 * audience map" referenced in the spec.
 */
export class Registry {
  readonly #byCode = new Map<string, Audience>();

  /** Record an audience, keyed by its code. */
  add(audience: Audience): void {
    this.#byCode.set(bytesToHex(audience.code), audience);
  }

  /** The audience for a raw audience code (e.g. a block's audience_code), if known. */
  byCode(code: Uint8Array): Audience | undefined {
    return this.#byCode.get(bytesToHex(code));
  }

  /** Number of audiences in the registry. */
  get size(): number {
    return this.#byCode.size;
  }
}
