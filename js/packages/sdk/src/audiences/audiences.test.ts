import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { bytesToHex, utf8ToBytes } from "../bytes.js";
import { openWorld, getAudienceCode } from "../derive/index.js";
import { openIdentity } from "../identity/index.js";
import { encryptForBlock, decryptData } from "../encryption/index.js";
import { newBlock, signBlock, type Block } from "../block/index.js";
import {
  publicAudience,
  publicAudienceID,
  publicAudienceSecret,
  inboxAudience,
  reservedPublicAudiences,
  isReservedPublic,
  RESERVED_PUBLIC_COUNT,
  Registry,
} from "./index.js";

const vectors = JSON.parse(
  readFileSync(
    fileURLToPath(new URL("../../../../../conformance/vectors.json", import.meta.url)),
    "utf8",
  ),
) as {
  world: { seed: string; audienceCode: string };
  identity: { world: string; passphrase: string };
  blockID: { timestamp: number };
  audiences: { publicAudienceSecret: string; inboxCode: string };
};

const world = openWorld(vectors.world.seed);
const hex = (b: Uint8Array) => bytesToHex(b);

describe("public audiences", () => {
  it("matches the shared vectors and the derive code", () => {
    const pub = publicAudience(world, 1);
    expect(pub.id).toBe("bpprotocol.org/v1/audience/public-1");
    expect(hex(pub.code)).toBe(hex(getAudienceCode(world, pub.id)));
    expect(hex(pub.code)).toBe(vectors.world.audienceCode); // public-1 == AudienceURN
    expect(hex(publicAudienceSecret(world, publicAudienceID(1)))).toBe(
      vectors.audiences.publicAudienceSecret,
    );
    expect(pub.secret).toBeDefined();
    expect(pub.secret!.length).toBe(32);
  });

  it("is world-scoped", () => {
    const a = publicAudience(openWorld("world-a"), 1);
    const b = publicAudience(openWorld("world-b"), 1);
    expect(hex(a.code)).not.toBe(hex(b.code));
    expect(hex(a.secret!)).not.toBe(hex(b.secret!));
  });

  it("has a reserved set of 16 distinct audiences", () => {
    const set = reservedPublicAudiences(world);
    expect(set.length).toBe(RESERVED_PUBLIC_COUNT);
    expect(new Set(set.map((a) => hex(a.code))).size).toBe(RESERVED_PUBLIC_COUNT);
    expect([
      isReservedPublic(1),
      isReservedPublic(16),
      isReservedPublic(0),
      isReservedPublic(17),
    ]).toEqual([true, true, false, false]);
  });
});

describe("inbox audiences", () => {
  it("matches the shared inbox-code vector and has no secret", () => {
    const addr = openIdentity(world, vectors.identity.passphrase).address;
    const inbox = inboxAudience(world, addr);
    expect(inbox.id).toBe(`bpprotocol.org/v1/audience/inbox/${addr}`);
    expect(hex(inbox.code)).toBe(vectors.audiences.inboxCode);
    expect(inbox.secret).toBeUndefined();
  });

  it("is address-specific and world-scoped", () => {
    const a = inboxAudience(world, "1111111111111111111111111111111111111111");
    const b = inboxAudience(world, "2222222222222222222222222222222222222222");
    expect(hex(a.code)).not.toBe(hex(b.code));
    expect(
      hex(inboxAudience(openWorld("other"), "1111111111111111111111111111111111111111").code),
    ).not.toBe(hex(a.code));
  });
});

describe("registry", () => {
  it("resolves a block's audience code to the audience", () => {
    const reg = new Registry();
    reg.add(publicAudience(world, 1));
    reg.add(publicAudience(world, 2));
    expect(reg.size).toBe(2);
    const pub = publicAudience(world, 1);
    expect(reg.byCode(pub.code)?.id).toBe(pub.id);
    expect(reg.byCode(utf8ToBytes("unknown_code____"))).toBeUndefined();
  });
});

describe("public-audience round-trip", () => {
  it("a member reads/writes; an outsider cannot", () => {
    const author = openIdentity(world, "author");
    const typeCode = getAudienceCode(world, "bpprotocol.org/v1/types/content.post"); // any code is fine here
    const pub = publicAudience(world, 1);
    const plaintext = utf8ToBytes("town square message");

    const data = encryptForBlock(
      pub.secret!,
      1,
      typeCode,
      pub.code,
      vectors.blockID.timestamp,
      plaintext,
    );
    const block: Block = newBlock(
      author.address,
      typeCode,
      pub.code,
      vectors.blockID.timestamp,
      data,
    );
    signBlock(block, world, author.mldsa);

    // Another member resolves the audience via a registry and decrypts.
    const reg = new Registry();
    reg.add(publicAudience(world, 1));
    const aud = reg.byCode(block.audienceCode);
    expect(aud).toBeDefined();
    expect(hex(decryptData(aud!.secret!, block))).toBe(hex(plaintext));

    // An outsider without the World seed derives a different secret and cannot decrypt.
    const outsider = openWorld("a-different-world");
    const wrong = publicAudienceSecret(outsider, publicAudienceID(1));
    expect(() => decryptData(wrong, block)).toThrow();
  });
});
