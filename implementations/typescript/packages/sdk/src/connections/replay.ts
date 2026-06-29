import { bytesToHex } from "../bytes.js";

/** Default acceptance window for handshake timestamps, in seconds. */
export const DEFAULT_REPLAY_WINDOW_SECONDS = 300;

export class ReplayError extends Error {
  constructor() {
    super("connections: replayed handshake nonce");
    this.name = "ReplayError";
  }
}
export class StaleError extends Error {
  constructor() {
    super("connections: handshake timestamp outside acceptance window");
    this.name = "StaleError";
  }
}

/**
 * Rejects replayed and stale connect.request blocks: a (target, nonce) pair may
 * be accepted only once, and only if its timestamp (unix seconds) is within
 * `windowSeconds` of the guard's clock. Not safe for concurrent use.
 */
export class ReplayGuard {
  readonly #seen = new Set<string>();
  readonly #windowSeconds: number;
  #now: () => number;

  /**
   * @param windowSeconds acceptance window (defaults to 300s)
   * @param now returns the current time in unix seconds (defaults to Date.now)
   */
  constructor(
    windowSeconds: number = DEFAULT_REPLAY_WINDOW_SECONDS,
    now: () => number = () => Math.floor(Date.now() / 1000),
  ) {
    this.#windowSeconds = windowSeconds > 0 ? windowSeconds : DEFAULT_REPLAY_WINDOW_SECONDS;
    this.#now = now;
  }

  /**
   * Validate a handshake's freshness and uniqueness, recording the nonce on
   * success. Throws StaleError if the timestamp is too far from now, or
   * ReplayError if the (target, nonce) pair was already seen.
   */
  check(target: string, nonce: Uint8Array, timestamp: number): void {
    const delta = this.#now() - timestamp;
    if (delta < -this.#windowSeconds || delta > this.#windowSeconds) throw new StaleError();
    const key = `${target}/${bytesToHex(nonce)}`;
    if (this.#seen.has(key)) throw new ReplayError();
    this.#seen.add(key);
  }
}
