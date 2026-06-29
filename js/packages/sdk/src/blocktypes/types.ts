import { fromBinary, type DescMessage, type Message } from "@bufbuild/protobuf";
import * as pb from "../blockpb/block_types_pb.js";
import { getTypeCode, codeHex, type World } from "../derive/index.js";
import type { Block } from "../block/index.js";

// Canonical block-type URNs under the bpprotocol.org/v1/types/ namespace.
export const TYPE_CONTENT_POST = "bpprotocol.org/v1/types/content.post";
export const TYPE_CONTENT_REACTION = "bpprotocol.org/v1/types/content.reaction";
export const TYPE_CONTENT_TAG_REQUEST = "bpprotocol.org/v1/types/content.tag.request";
export const TYPE_CONTENT_TAG_ACCEPT = "bpprotocol.org/v1/types/content.tag.accept";
export const TYPE_CONTENT_CHUNKED_MANIFEST = "bpprotocol.org/v1/types/content.chunked.manifest";
export const TYPE_CONTENT_CHUNKED_BLOCK = "bpprotocol.org/v1/types/content.chunked.block";
export const TYPE_CONTENT_GALLERY = "bpprotocol.org/v1/types/content.gallery";
export const TYPE_CONTENT_QUOTE = "bpprotocol.org/v1/types/content.quote";
export const TYPE_CONNECT_REQUEST = "bpprotocol.org/v1/types/connect.request";
export const TYPE_CONNECT_RESPONSE = "bpprotocol.org/v1/types/connect.response";
export const TYPE_CONNECT_IDENTITY = "bpprotocol.org/v1/types/connect.identity";
export const TYPE_CONNECT_ROTATE = "bpprotocol.org/v1/types/connect.rotate";
export const TYPE_CONNECT_CLOSE = "bpprotocol.org/v1/types/connect.close";
export const TYPE_IDENTITY = "bpprotocol.org/v1/types/identity";
export const TYPE_IDENTITY_BURN = "bpprotocol.org/v1/types/identity.burn";
export const TYPE_CHUNK_MANIFEST = "bpprotocol.org/v1/types/chunk.manifest";
export const TYPE_CHUNK_BLOCK = "bpprotocol.org/v1/types/chunk.block";
export const TYPE_RPC_REQUEST = "bpprotocol.org/v1/types/rpc.request";
export const TYPE_RPC_RESPONSE = "bpprotocol.org/v1/types/rpc.response";
export const TYPE_RPC_RENDER = "bpprotocol.org/v1/types/rpc.render";

export class UnknownTypeError extends Error {
  constructor(urn?: string) {
    super(`blocktypes: unknown block type${urn ? `: ${urn}` : ""}`);
    this.name = "UnknownTypeError";
  }
}

export class RenderNotTransportableError extends Error {
  constructor() {
    super("blocktypes: rpc.render is local-only and must not be transmitted");
    this.name = "RenderNotTransportableError";
  }
}

// Every core block type with the schema for its payload.
const CORE_TYPES: ReadonlyArray<readonly [string, DescMessage]> = [
  [TYPE_CONTENT_POST, pb.ContentPostSchema],
  [TYPE_CONTENT_REACTION, pb.ContentReactionSchema],
  [TYPE_CONTENT_TAG_REQUEST, pb.ContentTagRequestSchema],
  [TYPE_CONTENT_TAG_ACCEPT, pb.ContentTagAcceptSchema],
  [TYPE_CONTENT_CHUNKED_MANIFEST, pb.ContentChunkedManifestSchema],
  [TYPE_CONTENT_CHUNKED_BLOCK, pb.ContentChunkedBlockSchema],
  [TYPE_CONTENT_GALLERY, pb.ContentGallerySchema],
  [TYPE_CONTENT_QUOTE, pb.ContentQuoteSchema],
  [TYPE_CONNECT_REQUEST, pb.ConnectRequestSchema],
  [TYPE_CONNECT_RESPONSE, pb.ConnectResponseSchema],
  [TYPE_CONNECT_IDENTITY, pb.ConnectIdentitySchema],
  [TYPE_CONNECT_ROTATE, pb.ConnectRotateSchema],
  [TYPE_CONNECT_CLOSE, pb.ConnectCloseSchema],
  [TYPE_IDENTITY, pb.IdentityDeclareSchema],
  [TYPE_IDENTITY_BURN, pb.IdentityBurnSchema],
  [TYPE_CHUNK_MANIFEST, pb.ChunkManifestSchema],
  [TYPE_CHUNK_BLOCK, pb.ChunkBlockSchema],
  [TYPE_RPC_REQUEST, pb.RpcRequestSchema],
  [TYPE_RPC_RESPONSE, pb.RpcResponseSchema],
  [TYPE_RPC_RENDER, pb.RpcRenderSchema],
];

const schemaByURN = new Map<string, DescMessage>(CORE_TYPES);

/** The protobuf schema for a core type URN, if known. */
export function payloadSchema(typeURN: string): DescMessage | undefined {
  return schemaByURN.get(typeURN);
}

/**
 * Instantiate and decode the payload for a known type URN. Refuses rpc.render
 * (RenderNotTransportableError) since render blocks are local-only and must
 * never arrive over a transport.
 */
export function decodePayload(typeURN: string, plaintext: Uint8Array): Message {
  if (typeURN === TYPE_RPC_RENDER) throw new RenderNotTransportableError();
  const schema = schemaByURN.get(typeURN);
  if (!schema) throw new UnknownTypeError(typeURN);
  return fromBinary(schema, plaintext);
}

/**
 * Maps a World's type codes back to their canonical URNs, so an incoming
 * block's type_code can be dispatched to the right payload type.
 */
export class Resolver {
  readonly #byCode = new Map<string, string>();

  constructor(world: World) {
    for (const [urn] of CORE_TYPES) {
      this.#byCode.set(codeHex(getTypeCode(world, urn)), urn);
    }
  }

  /** The URN for a raw type code, if it is a known core type. */
  typeURN(code: Uint8Array): string | undefined {
    return this.#byCode.get(codeHex(code));
  }

  /**
   * Resolve a block's type and decode its (already-decrypted) payload, returning
   * the URN and message. Enforces the rpc.render guard.
   */
  decode(block: Block, plaintext: Uint8Array): { urn: string; message: Message } {
    const urn = this.typeURN(block.typeCode);
    if (!urn) throw new UnknownTypeError();
    return { urn, message: decodePayload(urn, plaintext) };
  }
}
