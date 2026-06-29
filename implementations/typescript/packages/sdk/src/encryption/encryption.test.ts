import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { bytesToHex, hexToBytes, utf8ToBytes } from "../bytes.js";
import { keccak256 } from "../crypto/index.js";
import { openWorld, getTypeCode, getAudienceCode } from "../derive/index.js";
import { openIdentity } from "../identity/index.js";
import { newBlock, type Block } from "../block/index.js";
import {
  contentKey,
  aad,
  sealWithNonce,
  encryptForBlock,
  decryptData,
  NONCE_SIZE,
} from "./index.js";

const vectors = JSON.parse(
  readFileSync(
    fileURLToPath(new URL("../../../../../../conformance/vectors.json", import.meta.url)),
    "utf8",
  ),
) as {
  world: { seed: string };
  blockID: { version: number; timestamp: number };
  aead: {
    secretSeed: string;
    nonce: string;
    plaintext: string;
    contentKey: string;
    aad: string;
    data: string;
  };
};

const world = openWorld(vectors.world.seed);
const typeCode = getTypeCode(world, "bpprotocol.org/v1/types/content.post");
const audCode = getAudienceCode(world, "bpprotocol.org/v1/audience/public-1");
const secret = keccak256(utf8ToBytes(vectors.aead.secretSeed));
const nonce = utf8ToBytes(vectors.aead.nonce);
const plaintext = utf8ToBytes(vectors.aead.plaintext);
const VERSION = vectors.blockID.version;
const TIMESTAMP = vectors.blockID.timestamp;

const author = openIdentity(world, "author");

// Build a block carrying a given data field (id is derived but unused here).
const blockWith = (data: Uint8Array): Block =>
  newBlock(author.address, typeCode, audCode, TIMESTAMP, data);

const hex = (b: Uint8Array) => bytesToHex(b);

describe("AEAD derivations match the shared vectors", () => {
  it("content key and AAD", () => {
    expect(hex(contentKey(secret, audCode, nonce))).toBe(vectors.aead.contentKey);
    expect(hex(aad(VERSION, typeCode, audCode, TIMESTAMP))).toBe(vectors.aead.aad);
  });
});

describe("cross-impl ciphertext (Go ⇄ Node)", () => {
  it("Node reproduces the Go ciphertext byte-for-byte (Node → Go)", () => {
    const aadBytes = aad(VERSION, typeCode, audCode, TIMESTAMP);
    expect(hex(sealWithNonce(secret, audCode, nonce, plaintext, aadBytes))).toBe(vectors.aead.data);
  });

  it("Node decrypts the Go ciphertext (Go → Node)", () => {
    const got = decryptData(secret, blockWith(hexToBytes(vectors.aead.data)));
    expect(hex(got)).toBe(hex(plaintext));
  });
});

describe("round-trip and tamper resistance", () => {
  it("encrypt → decrypt round-trips with a fresh nonce", () => {
    const data = encryptForBlock(secret, VERSION, typeCode, audCode, TIMESTAMP, plaintext);
    expect(data.length).toBeGreaterThan(NONCE_SIZE);
    expect(hex(decryptData(secret, blockWith(data)))).toBe(hex(plaintext));
  });

  it("a wrong secret cannot decrypt", () => {
    const data = encryptForBlock(secret, VERSION, typeCode, audCode, TIMESTAMP, plaintext);
    const wrong = keccak256(utf8ToBytes("not the right secret"));
    expect(() => decryptData(wrong, blockWith(data))).toThrow();
  });

  it("tampering any AAD-bound field fails decryption", () => {
    const mutators: Array<[string, (b: Block) => void]> = [
      ["version", (b) => (b.version += 1)],
      ["typeCode", (b) => (b.typeCode[0] ^= 0xff)],
      ["audienceCode", (b) => (b.audienceCode[0] ^= 0xff)],
      ["timestamp", (b) => (b.timestamp += 1n)],
      ["ciphertext", (b) => (b.data[b.data.length - 1] ^= 0xff)],
      ["nonce", (b) => (b.data[0] ^= 0xff)],
    ];
    for (const [name, mutate] of mutators) {
      const b = blockWith(
        encryptForBlock(secret, VERSION, typeCode, audCode, TIMESTAMP, plaintext),
      );
      mutate(b);
      expect(() => decryptData(secret, b), name).toThrow();
    }
  });
});
