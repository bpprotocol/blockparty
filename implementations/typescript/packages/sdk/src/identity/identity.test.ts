import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { bytesToHex, utf8ToBytes } from "../bytes.js";
import { keccak256, signMldsa, verifyMldsa, encapsulate, decapsulate } from "../crypto/index.js";
import { openWorld, bytesToAddress } from "../derive/index.js";
import { openIdentity } from "./index.js";

const vectors = JSON.parse(
  readFileSync(
    fileURLToPath(new URL("../../../../../../conformance/vectors.json", import.meta.url)),
    "utf8",
  ),
) as {
  identity: {
    world: string;
    passphrase: string;
    address: string;
    mldsaPubKeccak: string;
    kyberPubKeccak: string;
  };
};

const hex = (b: Uint8Array) => bytesToHex(b);

describe("identity", () => {
  const v = vectors.identity;

  it("matches the shared identity vector (address + key fingerprints)", () => {
    const id = openIdentity(openWorld(v.world), v.passphrase);
    expect(id.address).toBe(v.address);
    expect(hex(keccak256(id.mldsa.publicKey))).toBe(v.mldsaPubKeccak);
    expect(hex(keccak256(id.kyber.publicKey))).toBe(v.kyberPubKeccak);
  });

  it("is deterministic and world-scoped", () => {
    const w = openWorld(v.world);
    expect(openIdentity(w, v.passphrase).address).toBe(openIdentity(w, v.passphrase).address);
    expect(openIdentity(openWorld("world-a"), v.passphrase).address).not.toBe(
      openIdentity(openWorld("world-b"), v.passphrase).address,
    );
  });

  it("is passphrase-sensitive", () => {
    const w = openWorld(v.world);
    expect(openIdentity(w, "alice").address).not.toBe(openIdentity(w, "bob").address);
  });

  it("produces usable keys bound to the address", () => {
    const id = openIdentity(openWorld(v.world), v.passphrase);

    const msg = utf8ToBytes("nothing can stop the signal");
    const sig = signMldsa(id.mldsa.secretKey, msg);
    expect(verifyMldsa(id.mldsa.publicKey, msg, sig)).toBe(true);

    const { ciphertext, sharedSecret } = encapsulate(id.kyber.publicKey);
    expect(hex(decapsulate(ciphertext, id.kyber.secretKey))).toBe(hex(sharedSecret));

    expect(bytesToAddress(id.mldsa.publicKey, id.kyber.publicKey)).toBe(id.address);
  });
});
