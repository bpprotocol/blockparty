import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { bytesToHex, hexToBytes, utf8ToBytes } from "../bytes.js";
import { openWorld, getTypeCode } from "../derive/index.js";
import { openIdentity } from "../identity/index.js";
import { encryptForBlock, decryptData } from "../encryption/index.js";
import { newBlock, signBlock, verifyBlock } from "../block/index.js";
import {
  connectionSecret0,
  audienceCode,
  rotatedSecret,
  startRequest,
  acceptRequest,
  peerFromIdentity,
  buildClose,
  ReplayGuard,
  ReplayError,
  StaleError,
  RotateMismatchError,
  WrongTargetError,
  type Connection,
} from "./index.js";

const vectors = JSON.parse(
  readFileSync(
    fileURLToPath(new URL("../../../../../sdk/vectors/vectors.json", import.meta.url)),
    "utf8",
  ),
) as {
  world: { seed: string };
  connection: {
    ss1: string;
    ss2: string;
    nonce: string;
    nonceResponse: string;
    ss3: string;
    ct3: string;
    connectionSecret0: string;
    audienceCode0: string;
    audienceCode1: string;
  };
};

const hex = (b: Uint8Array) => bytesToHex(b);
const world = openWorld(vectors.world.seed);

describe("connection derivations match the shared vectors", () => {
  it("connectionSecret0, audienceCode0, and audienceCode1 (after rotation)", () => {
    const c = vectors.connection;
    const cs0 = connectionSecret0(
      hexToBytes(c.ss1),
      hexToBytes(c.ss2),
      hexToBytes(c.nonce),
      hexToBytes(c.nonceResponse),
    );
    expect(hex(cs0)).toBe(c.connectionSecret0);
    expect(hex(audienceCode(cs0, 0))).toBe(c.audienceCode0);
    const cs1 = rotatedSecret(cs0, hexToBytes(c.ss3), hexToBytes(c.ct3), 1);
    expect(hex(audienceCode(cs1, 1))).toBe(c.audienceCode1);
  });
});

function handshake(): { alice: Connection; bob: Connection } {
  const alice = openIdentity(world, "alice");
  const bob = openIdentity(world, "bob");
  const { pending, request } = startRequest(alice, peerFromIdentity(bob));
  const { response, connection: bobConn } = acceptRequest(
    bob,
    peerFromIdentity(alice),
    request,
    "request-block-id",
  );
  return { alice: pending.complete(response), bob: bobConn };
}

describe("handshake", () => {
  it("both parties derive the same private audience secret and code", () => {
    const { alice, bob } = handshake();
    expect(hex(alice.audienceSecret())).toBe(hex(bob.audienceSecret()));
    expect(hex(alice.audienceCode())).toBe(hex(bob.audienceCode()));
    expect(alice.epoch).toBe(0);
    expect(bob.epoch).toBe(0);
  });

  it("rejects a request whose target is not this identity", () => {
    const alice = openIdentity(world, "alice");
    const bob = openIdentity(world, "bob");
    const { request } = startRequest(alice, peerFromIdentity(bob));
    request.target = alice.address; // not Bob
    expect(() => acceptRequest(bob, peerFromIdentity(alice), request, "id")).toThrow(
      WrongTargetError,
    );
  });
});

describe("private-audience exchange over a connection", () => {
  it("encrypt on one side, decrypt on the other", () => {
    const { alice, bob } = handshake();
    const author = openIdentity(world, "alice");
    const typeCode = getTypeCode(world, "bpprotocol.org/v1/types/connect.identity");
    const ts = 1700000000;
    const plaintext = utf8ToBytes('{"name":"Alice"}');

    const data = encryptForBlock(
      alice.audienceSecret(),
      1,
      typeCode,
      alice.audienceCode(),
      ts,
      plaintext,
    );
    const block = newBlock(author.address, typeCode, alice.audienceCode(), ts, data);
    signBlock(block, world, author.mldsa);

    expect(verifyBlock(block, world, author.mldsa.publicKey)).toBe(true);
    expect(hex(decryptData(bob.audienceSecret(), block))).toBe(hex(plaintext));
  });
});

describe("rotation", () => {
  it("produces a fresh, independent epoch on which both parties agree", () => {
    const { alice, bob } = handshake();
    const secret0 = hex(alice.audienceSecret());
    const code0 = hex(alice.audienceCode());

    const rot = alice.rotate();
    bob.applyRotate(rot);

    expect(alice.epoch).toBe(1);
    expect(bob.epoch).toBe(1);
    expect(hex(alice.audienceSecret())).toBe(hex(bob.audienceSecret()));
    expect(hex(alice.audienceSecret())).not.toBe(secret0);
    expect(hex(alice.audienceCode())).not.toBe(code0);
    expect(codeMatches(alice.audienceCode(), rot.newChannel)).toBe(true);
  });

  it("rejects a rotation whose channel does not match", () => {
    const { alice, bob } = handshake();
    const rot = alice.rotate();
    rot.newChannel = "deadbeefdeadbeefdeadbeefdeadbeef";
    expect(() => bob.applyRotate(rot)).toThrow(RotateMismatchError);
  });
});

function codeMatches(code: Uint8Array, hexStr: string): boolean {
  return bytesToHex(code) === hexStr;
}

describe("ReplayGuard", () => {
  it("rejects replayed nonces and stale timestamps", () => {
    const ts = 1700000000;
    const guard = new ReplayGuard(300, () => ts);
    const target = "cff62cff35a0cc4271c262c07a86cbccc63ad13a";
    const nonce = new Uint8Array(32).fill(0xab);

    expect(() => guard.check(target, nonce, ts)).not.toThrow();
    expect(() => guard.check(target, nonce, ts)).toThrow(ReplayError);
    expect(() => guard.check(target, new Uint8Array(32).fill(0xcd), ts - 1000)).toThrow(StaleError);
    // Same nonce, different target is independent.
    expect(() => guard.check("0".repeat(40), nonce, ts)).not.toThrow();
  });
});

describe("buildClose", () => {
  it("builds a close payload", () => {
    const c = buildClose("abc", "rotated_keys");
    expect(c.target).toBe("abc");
    expect(c.reason).toBe("rotated_keys");
  });
});
