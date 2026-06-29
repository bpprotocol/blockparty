import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { create, toBinary } from "@bufbuild/protobuf";
import {
  RpcRenderSchema,
  RpcRequestSchema,
  ChunkManifestSchema,
} from "../blockpb/block_types_pb.js";
import { bytesToHex, utf8ToBytes } from "../bytes.js";
import { keccak256, sha512 } from "../crypto/index.js";
import { openWorld, getTypeCode, getAudienceCode } from "../derive/index.js";
import { openIdentity } from "../identity/index.js";
import { newBlock } from "../block/index.js";
import {
  Resolver,
  decodePayload,
  payloadSchema,
  UnknownTypeError,
  RenderNotTransportableError,
  reassembleChunks,
  reassembleChunkManifest,
  ChecksumError,
  MissingChunkError,
  buildBurn,
  verifyBurn,
  TrustState,
  buildPing,
  handlePing,
  parsePingResponse,
  NotPingError,
  METHOD_CORE_PING,
  TYPE_CONTENT_POST,
  TYPE_RPC_RENDER,
} from "./index.js";

const vectors = JSON.parse(
  readFileSync(
    fileURLToPath(new URL("../../../../../conformance/vectors.json", import.meta.url)),
    "utf8",
  ),
) as {
  world: { seed: string };
  burn: {
    passphrase: string;
    identity: string;
    mldsaSkKeccak: string;
    kyberSkKeccak: string;
    burnNotice: string;
  };
};

const world = openWorld(vectors.world.seed);
const hex = (b: Uint8Array) => bytesToHex(b);

// Every core type URN (rpc.render is handled separately).
const ALL_URNS = [
  "content.post",
  "content.reaction",
  "content.tag.request",
  "content.tag.accept",
  "content.chunked.manifest",
  "content.chunked.block",
  "content.gallery",
  "content.quote",
  "connect.request",
  "connect.response",
  "connect.identity",
  "connect.rotate",
  "connect.close",
  "identity",
  "identity.burn",
  "chunk.manifest",
  "chunk.block",
  "rpc.request",
  "rpc.response",
].map((t) => `bpprotocol.org/v1/types/${t}`);

describe("type registry & resolver", () => {
  it("round-trips every core payload type", () => {
    for (const urn of ALL_URNS) {
      const schema = payloadSchema(urn)!;
      const bytes = toBinary(schema, create(schema, {}));
      const msg = decodePayload(urn, bytes);
      expect(msg.$typeName).toBe(schema.typeName);
    }
  });

  it("maps a block's type code back to its URN", () => {
    const r = new Resolver(world);
    expect(r.typeURN(getTypeCode(world, TYPE_CONTENT_POST))).toBe(TYPE_CONTENT_POST);
    expect(r.typeURN(utf8ToBytes("not-a-real-code_"))).toBeUndefined();
  });

  it("rejects unknown types", () => {
    expect(() => decodePayload("example.com/v1/types/unknown", new Uint8Array())).toThrow(
      UnknownTypeError,
    );
  });
});

describe("rpc.render guard", () => {
  const renderBytes = toBinary(RpcRenderSchema, create(RpcRenderSchema, { id: "r", method: "x" }));

  it("refuses a render payload over a transport", () => {
    expect(() => decodePayload(TYPE_RPC_RENDER, renderBytes)).toThrow(RenderNotTransportableError);

    const r = new Resolver(world);
    const author = openIdentity(world, "author");
    const block = newBlock(
      author.address,
      getTypeCode(world, TYPE_RPC_RENDER),
      getAudienceCode(world, "a"),
      1700000000,
      renderBytes,
    );
    expect(() => r.decode(block, renderBytes)).toThrow(RenderNotTransportableError);
  });
});

describe("chunk reassembly", () => {
  it("reassembles in order and verifies SHA-512", () => {
    const parts = new Map([
      ["c1", utf8ToBytes("Hello, ")],
      ["c2", utf8ToBytes("world")],
      ["c3", utf8ToBytes("!")],
    ]);
    const full = utf8ToBytes("Hello, world!");
    const sum = hex(sha512(full));
    expect(hex(reassembleChunks(["c1", "c2", "c3"], parts, sum))).toBe(hex(full));

    const m = create(ChunkManifestSchema, { chunks: ["c1", "c2", "c3"], sha512: sum });
    expect(hex(reassembleChunkManifest(m, parts))).toBe(hex(full));
  });

  it("rejects a bad checksum and a missing chunk", () => {
    const parts = new Map([["c1", utf8ToBytes("data")]]);
    expect(() => reassembleChunks(["c1"], parts, "00")).toThrow(ChecksumError);
    expect(() => reassembleChunks(["missing"], new Map(), "x")).toThrow(MissingChunkError);
  });
});

describe("identity.burn", () => {
  it("a Node burn matches the Go fixture and verifies (cross-impl)", () => {
    const id = openIdentity(world, vectors.burn.passphrase);
    const burn = buildBurn(id, vectors.burn.burnNotice);
    expect(burn.identity).toBe(vectors.burn.identity);
    // The revealed secret keys are byte-identical to the Go reference's.
    expect(hex(keccak256(burn.revealedMlDsa))).toBe(vectors.burn.mldsaSkKeccak);
    expect(hex(keccak256(burn.revealedKyber))).toBe(vectors.burn.kyberSkKeccak);
    expect(verifyBurn(burn)).toBe(true);
  });

  it("a forged burn is rejected and never flips trust state", () => {
    const id = openIdentity(world, vectors.burn.passphrase);
    const forged = buildBurn(id, "voluntary");
    forged.identity = "0".repeat(40); // claim a different address
    expect(verifyBurn(forged)).toBe(false);

    const ts = new TrustState();
    expect(ts.isBurned(id.address)).toBe(false);
    expect(ts.recordBurn(buildBurn(id, "voluntary"))).toBe(true);
    expect(ts.isBurned(id.address)).toBe(true);
    expect(ts.recordBurn(forged)).toBe(false);
  });
});

describe("core.ping", () => {
  it("build / handle / parse round-trips", () => {
    const req = buildPing("ping-1", 1746396000);
    expect(req.method).toBe(METHOD_CORE_PING);
    expect(req.params["timestamp"]).toBe("1746396000");

    const resp = handlePing(req, 1746396002, 1746396003, "Reference Node 1.0.0");
    expect(resp.id).toBe(req.id);
    const r = parsePingResponse(resp);
    expect(r).toEqual({
      timestamp: 1746396000,
      receivedAt: 1746396002,
      serverTime: 1746396003,
      node: "Reference Node 1.0.0",
    });
  });

  it("rejects a non-ping request", () => {
    const other = create(RpcRequestSchema, { id: "x", method: "example.com/v1/rpc/other" });
    expect(() => handlePing(other, 0, 0, "")).toThrow(NotPingError);
  });
});
