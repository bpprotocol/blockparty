import { concatBytes, utf8ToBytes } from "../bytes.js";
import type { Block } from "../blockpb/block_pb.js";

// Domain separators for the three signature layers. Including a distinct domain
// in each preimage prevents a signature from one layer being replayed as
// another. These, the field order, and the framing must match the Go reference
// (sdk/block/preimage.go) byte-for-byte so signatures verify across clients.
const WORLD_DOMAIN = "bpprotocol.org/v1/sig/world";
const AUTHOR_DOMAIN = "bpprotocol.org/v1/sig/author";
const COSIGN_DOMAIN = "bpprotocol.org/v1/sig/cosign";

const EMPTY = new Uint8Array(0);

/** 8-byte big-endian length prefix, then the field bytes. */
function appendField(parts: Uint8Array[], f: Uint8Array): void {
  const len = new Uint8Array(8);
  new DataView(len.buffer).setBigUint64(0, BigInt(f.length), false);
  parts.push(len, f);
}

/** Domain tag, then each field, every element length-prefixed. */
function preimage(domain: string, fields: Uint8Array[]): Uint8Array {
  const parts: Uint8Array[] = [];
  appendField(parts, utf8ToBytes(domain));
  for (const f of fields) appendField(parts, f);
  return concatBytes(...parts);
}

function u32(v: number): Uint8Array {
  const b = new Uint8Array(4);
  new DataView(b.buffer).setUint32(0, v >>> 0, false);
  return b;
}

function i64(v: bigint): Uint8Array {
  const b = new Uint8Array(8);
  new DataView(b.buffer).setBigUint64(0, BigInt.asUintN(64, v), false);
  return b;
}

function metaFields(b: Block): Uint8Array[] {
  return [u32(b.version), b.id, b.typeCode, b.audienceCode, i64(b.timestamp), b.data];
}

const worldSig = (b: Block): Uint8Array => b.sigs?.world ?? EMPTY;
const authorSig = (b: Block): Uint8Array => b.sigs?.author ?? EMPTY;

/** Signed by the World key: the core fields. */
export function worldPreimage(b: Block): Uint8Array {
  return preimage(WORLD_DOMAIN, metaFields(b));
}

/** Signed by the author: the core fields plus world_sig. */
export function authorPreimage(b: Block): Uint8Array {
  return preimage(AUTHOR_DOMAIN, [...metaFields(b), worldSig(b)]);
}

/** Signed by a co-signer: the core fields plus world_sig and author_sig. */
export function coSignPreimage(b: Block): Uint8Array {
  return preimage(COSIGN_DOMAIN, [...metaFields(b), worldSig(b), authorSig(b)]);
}
