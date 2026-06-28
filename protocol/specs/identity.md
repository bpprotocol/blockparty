---
Title: Identity Specification
Version: 0.1.0
Last Updated: 2026-06-28
Status: Draft
License: CC0
Canonical URL: https://bpprotocol.org/specs/identity
---

# 🧬 Identity Specification

An `Identity` is the cryptographic core of a participant in the BlockParty Protocol. It serves as a stable, deterministic anchor for signing data and encrypting messages within a World. Identities are **portable and regenerable**: given the same passphrase and World, the identical keypairs and address are reproduced on any device.

## 🔐 Structure

```ts
Identity {
  address: string               // Canonical 40-char hex identifier
  dilithiumKey: KeyPair         // Post-quantum signature key (author_sig)
  kyberKey: KeyPair             // Post-quantum KEM key (encryption / connections)
}
```

## 🥮 Derivation

An identity is derived from a user **passphrase** scoped to a **World**. The World's `WalletSalt` (see [Derivations](./derivations.md) §"World Key + Salts") namespaces the derivation so the same passphrase yields a *different* identity in each World — preserving World isolation. Each key uses a domain-separated seed so the signature and KEM keys are independent.

```ts
function OpenIdentity(world, passphrase):
    // World-scoped secret: same passphrase → different identity per World
    worldPassword = HKDF_SHA256(ikm=passphrase, salt=world.WalletSalt,
                                info="bpprotocol.org/v1/identity", len=32)

    dilithiumKey = MakeDilithiumPair(HKDF_SHA256(worldPassword, salt="", info="dilithium", len=32))
    kyberKey     = MakeKyberPair(HKDF_SHA256(worldPassword, salt="", info="mlkem", len=32))

    address      = BytesToAddress(dilithiumKey.pub, kyberKey.pub)   // see Derivations
    return { address, dilithiumKey, kyberKey }
```

- `BytesToAddress` is the single canonical address derivation defined in [Derivations](./derivations.md) — `hex(Keccak256("v1" || dilithiumKey.pub || kyberKey.pub)[-20:])`, a 40-character hex address binding **both** public keys. This spec, [Derivations](./derivations.md), and the [Whitepaper](../whitepaper.md) §4 all refer to that one definition.
- `worldPassword` consumes the World's `WalletSalt`, the salt previously derived but unused in earlier drafts.
- The `"dilithium:"` / `"mlkem:"` domain separation matches the [Whitepaper](../whitepaper.md) §4 pseudocode; the two specs are aligned.

> 🚧 Notes:
> - All key derivation functions must be deterministic and reproducible (see `DeterministicRNG` in [Derivations](./derivations.md)).
> - The `"v1"` address prefix allows the address scheme to evolve without colliding with future versions.

## 🔑 Use Cases

- Signing blocks (`author_sig`) and verifying authorship — via `dilithiumKey`
- Establishing private [connections](./connections.md) and decrypting audience content — via `kyberKey`
- Anchoring an [identity burn](./identity-burn-rfc.md): exposing the identity's root private keys induces the post-truth state

## ♻️ Lifecycle

- **Create / Recover:** run `OpenIdentity(world, passphrase)`; publish an [`identity`](./block-types.md) block to announce the address and public keys.
- **Connect:** establish private audiences via the [connection handshake](./connections.md).
- **Rotate:** advance a connection's keys with [`connect.rotate`](./connections.md) without abandoning the identity.
- **Burn:** expose the root private keys via [`identity.burn`](./identity-burn-rfc.md) to repudiate the identity.

## 🔗 Relationship to Persona

A `Persona` wraps an `Identity` and includes human-readable metadata (e.g. name, avatar) and application-layer keys. The protocol layer is only concerned with `Identity`.

