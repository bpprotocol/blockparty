// Audience-scoped AEAD (issue #15), mirroring ../../../../sdk/encryption:
// ML-KEM768-derived audience secret -> HKDF-SHA256 content key ->
// XChaCha20-Poly1305 over `data`, with block metadata bound as AAD.
export * from "./encryption.js";
