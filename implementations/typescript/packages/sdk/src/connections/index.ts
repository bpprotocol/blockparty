// Connection handshake (issue #17), mirroring ../../../../sdk/connections: a
// two-message ML-KEM768 exchange yielding a private audience, plus rotation,
// teardown, and replay protection. The pure per-epoch derivations are exported
// for conformance; the live handshake is non-deterministic.
export * from "./derive.js";
export * from "./connections.js";
export * from "./replay.js";
