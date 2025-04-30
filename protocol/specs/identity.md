---
Title: Identity Specification
Version: 0.1.0
Last Updated: 2025-04-24
Status: Draft
License: CC0
Canonical URL: https://bpprotocol.org/specs/identity
---

# 🧬 Identity Specification

An `Identity` is the cryptographic core of a participant in the BlockParty Protocol. It serves as a stable, deterministic anchor for signing data and encrypting messages across all worlds.

## 🔐 Structure

```ts
Identity {
  address: string               // Canonical identifier
  dilithiumKey: KeyPair         // Post-quantum signature key
  kyberKey: KeyPair             // Post-quantum encryption key
}
```

## 🥮 Derivation

Identities are derived deterministically from a seed value (e.g. a Keccak-256 digest of user input).

```ts
function makeIdentity(seed):
    dilithiumKey = MakeDilithiumPair(seed)
    kyberKey = MakeKyberPair(seed)
    address = Keccak256("v1" || dilithiumKey.pub || kyberKey.pub)
    return { address, dilithiumKey, kyberKey }
```

> 🚧 Notes:
> - `address` format may be revised in future versions to support versioning or encoding changes.
> - All key derivation functions must be deterministic and reproducible.

## 🔑 Use Cases

- Signing blocks or messages
- Verifying authorship of published content
- Encrypting or decrypting messages in private audiences (via `kyberKey`)

## 🔗 Relationship to Persona

A `Persona` wraps an `Identity` and includes human-readable metadata (e.g. name, avatar) and application-layer keys. The protocol layer is only concerned with `Identity`.

