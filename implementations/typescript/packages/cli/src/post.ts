import {
  create,
  toBinary,
  utf8ToBytes,
  getTypeCode,
  newBlock,
  signBlock,
  encryptForBlock,
  decryptData,
  decodePayload,
  ContentPostSchema,
  TYPE_CONTENT_POST,
  type World,
  type Identity,
  type Resolver,
  type Block,
  type ContentPost,
} from "@blockparty/sdk";

const CONTENT_TYPE_HEADER = "bpprotocol.org/v1/content-type";
const decoder = new TextDecoder();

/**
 * Build a signed content.post block whose text body is encrypted to an audience
 * (identified by code + secret) within a World.
 */
export function buildPost(
  world: World,
  author: Identity,
  audienceCode: Uint8Array,
  audienceSecret: Uint8Array,
  timestamp: number,
  text: string,
): Block {
  const typeCode = getTypeCode(world, TYPE_CONTENT_POST);
  const payload = toBinary(
    ContentPostSchema,
    create(ContentPostSchema, {
      headers: { [CONTENT_TYPE_HEADER]: "text/plain" },
      body: utf8ToBytes(text),
    }),
  );
  const data = encryptForBlock(audienceSecret, 1, typeCode, audienceCode, timestamp, payload);
  const block = newBlock(author.address, typeCode, audienceCode, timestamp, data);
  signBlock(block, world, author.mldsa);
  return block;
}

/**
 * Decrypt and decode a content.post block with a known audience secret,
 * returning the text body, or undefined if it is not a content.post for this
 * World. Decryption is authenticated, so a tampered block throws.
 */
export function openPost(
  resolver: Resolver,
  block: Block,
  audienceSecret: Uint8Array,
): string | undefined {
  if (resolver.typeURN(block.typeCode) !== TYPE_CONTENT_POST) return undefined;
  const plaintext = decryptData(audienceSecret, block);
  const post = decodePayload(TYPE_CONTENT_POST, plaintext) as ContentPost;
  return decoder.decode(post.body);
}
