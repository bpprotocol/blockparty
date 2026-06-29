import { hkdfSha256 } from "../crypto/index.js";
import { getAudienceCode, codeHex, type World } from "../derive/index.js";
import { utf8ToBytes } from "../bytes.js";

/** Namespace for the well-known public audiences. */
export const PUBLIC_AUDIENCE_PREFIX = "bpprotocol.org/v1/audience/public-";
/** Namespace for per-identity inbox audiences. */
export const INBOX_AUDIENCE_PREFIX = "bpprotocol.org/v1/audience/inbox/";
/** Number of reserved well-known public audiences (public-1 .. public-16). */
export const RESERVED_PUBLIC_COUNT = 16;

const AUDIENCE_SECRET_INFO = "bpprotocol.org/v1/audience-secret";

/**
 * A derived audience: its identifier, world-scoped code (addressing label), and
 * — for confidential audiences — its 32-byte secret. Rendezvous-only audiences
 * (e.g. inbox) have no secret.
 */
export interface Audience {
  readonly id: string;
  readonly code: Uint8Array;
  readonly secret?: Uint8Array;
}

/** The canonical identifier for public audience n. */
export function publicAudienceID(n: number): string {
  return `${PUBLIC_AUDIENCE_PREFIX}${n}`;
}

/** Whether n is in the reserved well-known range. */
export function isReservedPublic(n: number): boolean {
  return Number.isInteger(n) && n >= 1 && n <= RESERVED_PUBLIC_COUNT;
}

/**
 * Derive the 32-byte secret for a public audience within a World. Per
 * encryption.md it is HKDF-SHA256 over the World's audience salt, salted by the
 * audience code (its hex form), with a fixed info. Only holders of the World
 * seed can derive it.
 */
export function publicAudienceSecret(world: World, audienceID: string): Uint8Array {
  const code = getAudienceCode(world, audienceID);
  return hkdfSha256(
    world.audienceSalt,
    utf8ToBytes(codeHex(code)),
    utf8ToBytes(AUDIENCE_SECRET_INFO),
    32,
  );
}

/** Derive the full public audience (code + secret) for public-n within a World. */
export function publicAudience(world: World, n: number): Audience {
  const id = publicAudienceID(n);
  return { id, code: getAudienceCode(world, id), secret: publicAudienceSecret(world, id) };
}

/** Derive all reserved public audiences (public-1 .. public-16) for a World. */
export function reservedPublicAudiences(world: World): Audience[] {
  const out: Audience[] = [];
  for (let n = 1; n <= RESERVED_PUBLIC_COUNT; n++) out.push(publicAudience(world, n));
  return out;
}

/** The rendezvous audience identifier for an address. */
export function inboxAudienceID(address: string): string {
  return `${INBOX_AUDIENCE_PREFIX}${address}`;
}

/**
 * Derive the rendezvous audience for delivering connection handshake blocks to
 * an address. It has a code but no secret: the inbox is a rendezvous, and
 * handshake confidentiality comes from the KEM, not the inbox.
 */
export function inboxAudience(world: World, address: string): Audience {
  const id = inboxAudienceID(address);
  return { id, code: getAudienceCode(world, id) };
}
