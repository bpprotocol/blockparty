import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { bytesToHex, utf8ToBytes } from "../bytes.js";
import { keccak256 } from "../crypto/index.js";
import { openWorld, getTypeCode, getAudienceCode } from "../derive/index.js";
import { openIdentity } from "../identity/index.js";
import {
  newBlock,
  signBlock,
  verifyBlock,
  addCoSignature,
  verifyCoSignature,
  coSignatures,
  encode,
  decode,
  idHex,
  type Block,
} from "./index.js";

const vectors = JSON.parse(
  readFileSync(
    fileURLToPath(new URL("../../../../../conformance/vectors.json", import.meta.url)),
    "utf8",
  ),
) as {
  world: { seed: string };
  block: {
    passphrase: string;
    typeURN: string;
    audienceURN: string;
    timestamp: number;
    data: string;
    idHex: string;
    encodedKeccak: string;
  };
};

const world = openWorld(vectors.world.seed);
const author = openIdentity(world, vectors.block.passphrase);
const typeCode = getTypeCode(world, vectors.block.typeURN);
const audCode = getAudienceCode(world, vectors.block.audienceURN);

function signed(data: Uint8Array): Block {
  const b = newBlock(author.address, typeCode, audCode, vectors.block.timestamp, data);
  signBlock(b, world, author.mldsa);
  return b;
}

describe("cross-impl block fixture (Go ⇄ Node)", () => {
  it("produces a byte-identical signed block to the Go reference", () => {
    const b = signed(utf8ToBytes(vectors.block.data));
    // Same block ID and same canonical encoding fingerprint as Go: since keygen,
    // deterministic ML-DSA signing, and canonical proto encoding all match, the
    // entire signed block is byte-identical across implementations.
    expect(idHex(b)).toBe(vectors.block.idHex);
    expect(bytesToHex(keccak256(encode(b)))).toBe(vectors.block.encodedKeccak);
    expect(verifyBlock(b, world, author.mldsa.publicKey)).toBe(true);
  });
});

describe("sign / verify", () => {
  it("verifies a freshly signed block", () => {
    expect(verifyBlock(signed(utf8ToBytes("hello")), world, author.mldsa.publicKey)).toBe(true);
  });

  it("rejects a missing signature and the wrong keys", () => {
    const unsigned = newBlock(author.address, typeCode, audCode, 1700000000, utf8ToBytes("x"));
    expect(verifyBlock(unsigned, world, author.mldsa.publicKey)).toBe(false);

    const b = signed(utf8ToBytes("hello"));
    const stranger = openIdentity(world, "stranger");
    expect(verifyBlock(b, world, stranger.mldsa.publicKey)).toBe(false);
    const otherWorld = openWorld("a-different-world");
    expect(verifyBlock(b, otherWorld, author.mldsa.publicKey)).toBe(false);
  });

  it("fails verification when any signed field is tampered", () => {
    const mutators: Array<[string, (b: Block) => void]> = [
      ["version", (b) => (b.version += 1)],
      ["id", (b) => (b.id[0] ^= 0xff)],
      ["typeCode", (b) => (b.typeCode[0] ^= 0xff)],
      ["audienceCode", (b) => (b.audienceCode[0] ^= 0xff)],
      ["timestamp", (b) => (b.timestamp += 1n)],
      ["data", (b) => (b.data[0] ^= 0xff)],
      ["world_sig", (b) => (b.sigs!.world[0] ^= 0xff)],
      ["author_sig", (b) => (b.sigs!.author[0] ^= 0xff)],
    ];
    for (const [name, mutate] of mutators) {
      const b = signed(utf8ToBytes("hello"));
      mutate(b);
      expect(verifyBlock(b, world, author.mldsa.publicKey), name).toBe(false);
    }
  });
});

describe("encode / decode", () => {
  it("round-trips canonically and preserves signatures + co-signatures", () => {
    const cosigner = openIdentity(world, "cosigner");
    const b = signed(utf8ToBytes("hello"));
    addCoSignature(b, "bpprotocol.org/v1/cosign.endorse", cosigner.mldsa);

    const enc = encode(b);
    const b2 = decode(enc);
    expect(bytesToHex(encode(b2))).toBe(bytesToHex(enc)); // canonical & stable
    expect(verifyBlock(b2, world, author.mldsa.publicKey)).toBe(true);
    const cos = coSignatures(b2);
    expect(cos.length).toBe(1);
    expect(verifyCoSignature(b2, cos[0]!, cosigner.mldsa.publicKey)).toBe(true);
  });
});

describe("co-signatures", () => {
  it("add/verify works; bad and tampered co-signatures are ignored", () => {
    const cosigner = openIdentity(world, "cosigner");
    const b = signed(utf8ToBytes("hello"));
    addCoSignature(b, "bpprotocol.org/v1/cosign.endorse", cosigner.mldsa);

    const cos = coSignatures(b);
    expect(verifyCoSignature(b, cos[0]!, cosigner.mldsa.publicKey)).toBe(true);
    // Wrong signer key.
    expect(verifyCoSignature(b, cos[0]!, author.mldsa.publicKey)).toBe(false);
    // A co-signature never affects the required signatures.
    expect(verifyBlock(b, world, author.mldsa.publicKey)).toBe(true);

    // Tampered co-signature fails its own check but the block stays valid.
    cos[0]!.sig[0] ^= 0xff;
    expect(verifyCoSignature(b, cos[0]!, cosigner.mldsa.publicKey)).toBe(false);
    expect(verifyBlock(b, world, author.mldsa.publicKey)).toBe(true);
  });
});
