import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import {
  BlockSchema,
  SignaturesSchema,
  ExtraSignatureSchema,
  type Block,
  type ExtraSignature,
} from "../blockpb/block_pb.js";
import { signMldsa, verifyMldsa } from "../crypto/index.js";
import { getBlockID, type World } from "../derive/index.js";
import type { KeyPair } from "../crypto/index.js";
import { bytesToHex, hexToBytes } from "../bytes.js";
import { authorPreimage, coSignPreimage, worldPreimage } from "./preimage.js";

export type { Block, ExtraSignature };

/** The numeric protocol version stamped on new blocks. */
export const VERSION = 1;

/**
 * Construct an unsigned block with its content-binding ID derived from the
 * author's address. Call signBlock next. `timestamp` is unix seconds.
 */
export function newBlock(
  authorAddress: string,
  typeCode: Uint8Array,
  audienceCode: Uint8Array,
  timestamp: number,
  data: Uint8Array,
): Block {
  const idHex = getBlockID(VERSION, timestamp, audienceCode, authorAddress, typeCode, data);
  return create(BlockSchema, {
    version: VERSION,
    id: hexToBytes(idHex),
    typeCode,
    audienceCode,
    timestamp: BigInt(timestamp),
    data,
  });
}

/** The block's ID as a lowercase hex string. */
export function idHex(b: Block): string {
  return bytesToHex(b.id);
}

/**
 * Apply the world signature (with the World's signing key) then the author
 * signature (with the author's signing key). Leaves co-signatures untouched.
 */
export function signBlock(b: Block, world: World, author: KeyPair): void {
  if (!b.sigs) b.sigs = create(SignaturesSchema, {});
  b.sigs.world = signMldsa(world.signingKey.secretKey, worldPreimage(b));
  b.sigs.author = signMldsa(author.secretKey, authorPreimage(b));
}

/**
 * Verify the two required signatures: world_sig against the World's signing key
 * and author_sig against authorPublicKey. Co-signatures are ignored, so a
 * missing or invalid co-signature never invalidates a block. Returns false if a
 * required signature is missing or invalid (e.g. any signed field was tampered).
 */
export function verifyBlock(b: Block, world: World, authorPublicKey: Uint8Array): boolean {
  if (!b.sigs || b.sigs.world.length === 0 || b.sigs.author.length === 0) return false;
  if (!verifyMldsa(world.signingKey.publicKey, worldPreimage(b), b.sigs.world)) return false;
  return verifyMldsa(authorPublicKey, authorPreimage(b), b.sigs.author);
}

/**
 * Append a co-signature of the given namespaced type, signed over the block's
 * fields plus its world and author signatures. The block must already be signed.
 */
export function addCoSignature(b: Block, type: string, signer: KeyPair): void {
  if (!b.sigs) b.sigs = create(SignaturesSchema, {});
  b.sigs.extra.push(
    create(ExtraSignatureSchema, {
      type,
      sig: signMldsa(signer.secretKey, coSignPreimage(b)),
    }),
  );
}

/** Whether extra is a valid co-signature of b by signerPublicKey. */
export function verifyCoSignature(
  b: Block,
  extra: ExtraSignature,
  signerPublicKey: Uint8Array,
): boolean {
  return verifyMldsa(signerPublicKey, coSignPreimage(b), extra.sig);
}

/** The block's co-signatures (sigs.extra), or an empty array. */
export function coSignatures(b: Block): ExtraSignature[] {
  return b.sigs?.extra ?? [];
}

/** Serialize a block to its canonical, deterministic Protobuf wire form. */
export function encode(b: Block): Uint8Array {
  return toBinary(BlockSchema, b);
}

/** Parse a block from its Protobuf wire form. */
export function decode(data: Uint8Array): Block {
  return fromBinary(BlockSchema, data);
}
