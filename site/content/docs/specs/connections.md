---
title: Connections
version: 0.1.0
updated: 2026-06-28
status: Draft
license: CC0
canonical: https://bpprotocol.org/specs/connections
---

# Connections

A **connection** is a private channel between two identities, backed by a post-quantum key agreement. Connecting yields a **private audience** whose secret only the two parties can derive; all subsequent confidential traffic is encrypted to that audience. This document specifies the `connect.*` handshake end-to-end, the derivation of the private audience, replay protection, rotation, and teardown.

Depends on: [Derivations](./derivations.md), [Block Encryption](./encryption.md), [Audiences](./audiences.md), [Block Types](./block-types.md).

---

## 🧷 Prerequisites

Each identity publishes an [`identity`](./block-types.md) block listing its post-quantum public keys (`dilithiumKey.pub`, `kyberKey.pub`) — see [Identity](./identity.md). To start a connection, the initiator must know the target's address and static `kyberKey.pub` (typically learned from the target's `identity` block on a [public audience](./audiences.md)).

### Inbox audience (rendezvous)

Handshake blocks are addressed to the recipient's **inbox audience** — a per-identity rendezvous label any World member can derive:

```ts
inboxAudienceID  = "bpprotocol.org/v1/audience/inbox/" + recipientAddress
inboxAudienceCode = GetAudienceCode(world, inboxAudienceID)
```

The inbox is a rendezvous, not a confidential channel: handshake payloads carry only KEM ciphertexts and random nonces, none of which are secret (an ML-KEM768 ciphertext is safe to publish — only the holder of the corresponding private key can decapsulate it). Confidentiality of the *connection* comes from the key agreement, not from the inbox. Handshake blocks MAY therefore be sent as [plaintext blocks](./encryption.md).

---

## 🤝 Handshake

Two messages establish a connection. Alice is the initiator, Bob the target.

### 1. `connect.request` (Alice → Bob's inbox)

```ts
ephA = MakeKyberPair(random_seed)                    // ephemeral KEM keypair (forward secrecy)
(ct1, ss1) = MLKEM768.Encaps(Bob.kyberPub)           // to Bob's STATIC key — only Bob can recover ss1
```

Payload:
```json
{
  "target":        "<bob_address>",
  "init_kem_pub":  "<base64 ephA.pub>",
  "kem_ciphertext":"<base64 ct1>",
  "nonce":         "<base64 random(32)>"
}
```

- `ss1` authenticates Bob: only the holder of Bob's static `kyberKey` private key can `Decaps(ct1)` to recover it.
- `ephA.pub` is a fresh ephemeral KEM key contributed for forward secrecy (see message 2).

### 2. `connect.response` (Bob → Alice's inbox)

Bob recovers `ss1 = MLKEM768.Decaps(ct1, Bob.kyberPriv)`, then contributes fresh entropy by encapsulating to Alice's **ephemeral** key:

```ts
(ct2, ss2) = MLKEM768.Encaps(ephA.pub)               // to Alice's EPHEMERAL key
```

Payload:
```json
{
  "request":        "<connect_request_block_id>",
  "kem_ciphertext": "<base64 ct2>",
  "nonce_response": "<base64 random(32)>"
}
```

Alice recovers `ss2 = MLKEM768.Decaps(ct2, ephA.priv)`. Both parties now hold `ss1` and `ss2`.

### 3. Derive the connection secret & private audience

```ts
// epoch starts at 0; bumped by connect.rotate
connectionSecret(0) = HKDF_SHA256(
    ikm  = ss1 || ss2,
    salt = nonce || nonce_response,
    info = "bpprotocol.org/v1/connect:epoch=0",
    len  = 32)

audienceSecret(e) = HKDF_SHA256(connectionSecret(e), salt="",
    info = "bpprotocol.org/v1/connect-audience:epoch=" + e, len = 32)

audienceCode(e)   = hex(Keccak256(audienceSecret(e))[0:CODE_BYTES])
```

- The private **audience secret** is `audienceSecret(e)`; it is exactly the "audience secret" consumed by [Block Encryption](./encryption.md) to derive per-block content keys.
- The private **audience code** is derived from the secret, so it is **unlinkable** to either participant's address — an outside observer cannot tell which two identities a private audience belongs to.
- Contributory agreement (both `ss1` and `ss2`) ensures neither party unilaterally controls the secret; the ephemeral `ephA` provides forward secrecy for `ss2`.

### 4. `connect.identity` (either party → private audience)

Once the private audience exists, each party reveals private metadata, **encrypted to `audienceCode(e)`**:

```json
{
  "name": "Alice",
  "keys": ["<additional_or_channel_pubkey>", "..."]
}
```

`keys` are any additional public keys the party wishes to share within the connection (e.g. application-layer or per-device keys). This is the first confidential message and confirms the channel works in both directions.

---

## 🛡️ Replay & Freshness

- `nonce` and `nonce_response` are 32 bytes of fresh randomness per handshake.
- A client MUST reject a `connect.request` whose `(target, nonce)` pair it has already seen, and SHOULD reject one whose block `timestamp` falls outside an acceptance window (default ±300 s) relative to its own clock, allowing for delay-tolerant transports via a configurable wider window.
- Because the connection secret mixes both nonces, a replayed handshake message cannot reconstruct a prior session's audience secret.

---

## 🔁 Rotation

`connect.rotate` advances the channel to a new epoch with **fresh entropy**, bounding the damage from a compromised epoch secret (a forward ratchet, not a mere counter bump).

```ts
(ct3, ss3) = MLKEM768.Encaps(peer.kyberPub)          // fresh contribution
connectionSecret(e+1) = HKDF_SHA256(
    ikm  = connectionSecret(e) || ss3,
    salt = ct3,                                       // the rotation ciphertext doubles as the salt
    info = "bpprotocol.org/v1/connect:epoch=" + (e+1),
    len  = 32)
```

`ct3` (the rotation's `kem_ciphertext`) is the HKDF salt: it is fresh per rotation and known to both parties (the rotator generates it, the recipient receives it in the payload), so no extra nonce field is needed.

`connect.rotate` payload (sent encrypted to the *current* `audienceCode(e)`):
```json
{
  "new_channel":    "<hex audienceCode(e+1)>",
  "kem_ciphertext": "<base64 ct3>"
}
```

- The recipient recovers `ss3`, derives `connectionSecret(e+1)`, and confirms `audienceCode(e+1)` matches `new_channel`.
- The old audience is abandoned: both parties stop publishing to and decrypting `audienceCode(e)`. Past blocks under the old epoch remain readable to anyone who already held that epoch's secret (no retroactive deletion — see [encryption.md](./encryption.md) §"Forward Secrecy & Rotation").
- Rotation also covers key-compromise recovery: rotating after a suspected leak re-keys the channel from new KEM entropy. Whole-identity repudiation is a separate mechanism — see [identity burn](./identity-burn-rfc.md).

---

## 🚪 Close

`connect.close` signals teardown:
```json
{
  "target": "<peer_address>",
  "reason": "rotated_keys"   // or "closed", "compromised", "expired", "other"
}
```

After close, both parties cease deriving and accepting that connection's audiences. Close is advisory: like everything in BlockParty it is not protocol-enforced, but well-behaved clients honor it.

---

## 🔄 End-to-End Summary

```
Alice                                              Bob
  | connect.request  → Bob.inbox                     |
  |   {init_kem_pub=ephA.pub, ct1=Encaps(Bob.kyber), nonce}
  |------------------------------------------------->| Decaps(ct1) → ss1
  |                                                  |
  |                    Bob.inbox ← connect.response  |
  |   {request, ct2=Encaps(ephA.pub), nonce_response}|
  |<-------------------------------------------------|
  | Decaps(ct2) → ss2                                |
  |                                                  |
  | both derive connectionSecret(0) → audienceSecret(0) → audienceCode(0)
  |                                                  |
  | connect.identity  ⇄  (encrypted to audienceCode(0))
  | ... confidential traffic (dm.message, content.*, etc.) ...
  | connect.rotate → audienceCode(1) ; connect.close to tear down
```

---

## 🔗 Related Specifications

- [Block Types](./block-types.md) — `connect.*` payload schemas & proto
- [Block Encryption](./encryption.md) — audience secret → content key
- [Audiences](./audiences.md) — public audiences & inbox rendezvous
- [Identity](./identity.md) — identity keys & address
