// @blockparty/sdk — the reusable BlockParty Protocol library.
//
// This is the scaffold (issue #11). The protocol modules land in subsequent
// issues, each mirroring the Go reference in ../../../sdk and validated against
// the shared conformance vectors (../../../sdk/vectors/vectors.json):
//
//   #12 crypto · #13 derivations/identity · #14 blocks · #15 encryption
//   #16 audiences · #17 connections · #18 block types
//
// Signature scheme: ML-DSA-65 (FIPS 204); KEM: ML-KEM768 (FIPS 203) — see #20.

/** The BlockParty protocol version this SDK targets. */
export const PROTOCOL_VERSION = "0.1.0";

/** A placeholder export so the package builds and is importable; replaced as the
 * protocol modules land. */
export function hello(): string {
  return `BlockParty SDK v${PROTOCOL_VERSION}`;
}
