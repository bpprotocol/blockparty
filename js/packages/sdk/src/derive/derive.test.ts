import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { bytesToHex, utf8ToBytes } from "../bytes.js";
import { keccak256 } from "../crypto/index.js";
import {
  openWorld,
  getTypeCode,
  getAudienceCode,
  bytesToAddress,
  getBlockID,
  CODE_BYTES,
} from "./index.js";
import { openIdentity } from "../identity/index.js";

const vectors = JSON.parse(
  readFileSync(
    fileURLToPath(new URL("../../../../../conformance/vectors.json", import.meta.url)),
    "utf8",
  ),
) as {
  world: {
    seed: string;
    walletSalt: string;
    typeSalt: string;
    audienceSalt: string;
    signingKeyKeccak: string;
    typeCode: string;
    audienceCode: string;
  };
  identity: { world: string; passphrase: string };
  blockID: { version: number; timestamp: number; data: string; id: string };
};

// The canonical inputs that produce world.typeCode / world.audienceCode (same
// as the Go vectors generator).
const TYPE_URN = "bpprotocol.org/v1/types/content.post";
const AUDIENCE_URN = "bpprotocol.org/v1/audience/public-1";

const hex = (b: Uint8Array) => bytesToHex(b);

describe("worlds", () => {
  it("matches the shared world vectors (salts + signing key)", () => {
    const w = openWorld(vectors.world.seed);
    expect(hex(w.walletSalt)).toBe(vectors.world.walletSalt);
    expect(hex(w.typeSalt)).toBe(vectors.world.typeSalt);
    expect(hex(w.audienceSalt)).toBe(vectors.world.audienceSalt);
    expect(hex(keccak256(w.signingKey.publicKey))).toBe(vectors.world.signingKeyKeccak);
  });

  it("is deterministic and seed-scoped", () => {
    expect(hex(openWorld("x").typeSalt)).toBe(hex(openWorld("x").typeSalt));
    expect(hex(openWorld("a").typeSalt)).not.toBe(hex(openWorld("b").typeSalt));
  });
});

describe("codes", () => {
  const w = openWorld(vectors.world.seed);

  it("matches the shared type/audience code vectors", () => {
    expect(hex(getTypeCode(w, TYPE_URN))).toBe(vectors.world.typeCode);
    expect(hex(getAudienceCode(w, AUDIENCE_URN))).toBe(vectors.world.audienceCode);
    expect(getTypeCode(w, TYPE_URN).length).toBe(CODE_BYTES);
  });

  it("is world-scoped", () => {
    expect(hex(getTypeCode(openWorld("a"), TYPE_URN))).not.toBe(
      hex(getTypeCode(openWorld("b"), TYPE_URN)),
    );
  });

  it("normalizes whitespace and trailing slashes", () => {
    const base = hex(getTypeCode(w, TYPE_URN));
    expect(hex(getTypeCode(w, `  ${TYPE_URN}  `))).toBe(base);
    expect(hex(getTypeCode(w, `${TYPE_URN}/`))).toBe(base);
  });
});

describe("address", () => {
  it("binds both keys, 40 hex chars, and is key-sensitive", () => {
    const dil = utf8ToBytes("ml-dsa-public-key-bytes");
    const kyb = utf8ToBytes("kyber-public-key-bytes");
    const addr = bytesToAddress(dil, kyb);
    expect(addr.length).toBe(40);
    const want = bytesToHex(keccak256(utf8ToBytes("v1"), dil, kyb).subarray(-20));
    expect(addr).toBe(want);
    expect(bytesToAddress(utf8ToBytes("x"), kyb)).not.toBe(addr);
    expect(bytesToAddress(dil, utf8ToBytes("y"))).not.toBe(addr);
  });
});

describe("block ID", () => {
  it("reproduces the shared block-ID vector and is content-binding", () => {
    const w = openWorld(vectors.world.seed);
    const id = openIdentity(w, vectors.identity.passphrase);
    const tc = getTypeCode(w, TYPE_URN);
    const ac = getAudienceCode(w, AUDIENCE_URN);

    const blockId = getBlockID(
      vectors.blockID.version,
      vectors.blockID.timestamp,
      ac,
      id.address,
      tc,
      utf8ToBytes(vectors.blockID.data),
    );
    expect(blockId).toBe(vectors.blockID.id);

    const other = getBlockID(
      vectors.blockID.version,
      vectors.blockID.timestamp,
      ac,
      id.address,
      tc,
      utf8ToBytes("hello!"),
    );
    expect(other).not.toBe(blockId);
    expect(blockId.length).toBe(64);
  });
});
