// @blockparty/sdk — the reusable BlockParty Protocol library.
//
// Modules land per issue, each mirroring the Go reference in ../../../sdk and
// validated against the shared conformance vectors
// (../../../sdk/vectors/vectors.json):
//
//   #12 crypto ✓ · #13 derivations/identity ✓ · #14 blocks ✓ · #15 encryption ✓
//   #16 audiences ✓ · #17 connections ✓ · #18 block types ✓
//
// Signature scheme: ML-DSA-65 (FIPS 204); KEM: ML-KEM768 (FIPS 203) — see #20.

/** The BlockParty protocol version this SDK targets. */
export const PROTOCOL_VERSION = "0.1.0";

export * from "./bytes.js";
export * from "./proto.js";
export * from "./crypto/index.js";
export * from "./derive/index.js";
export * from "./identity/index.js";
export * from "./block/index.js";
export * from "./encryption/index.js";
export * from "./audiences/index.js";
export * from "./connections/index.js";
export * from "./blocktypes/index.js";
// Generated payload message types and schemas (ContentPost, IdentityBurn, …).
export * from "./blockpb/block_types_pb.js";
