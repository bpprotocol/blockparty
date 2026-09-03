---
title: 'RFC: Burnable Identities and the Post-Truth State in Cryptographic Protocols'
version: 0.1.0
updated: 2026-06-28
status: Draft
license: CC0
canonical: https://bpprotocol.org/specs/identity-burn-rfc
---

# RFC: Burnable Identities and the Post-Truth State in Cryptographic Protocols

## 1. Abstract

This document proposes a cryptographic pattern known as the *burnable identity*, in which a user may deliberately reveal their root private key to induce a Post-Truth State. In this state, all prior signatures by that identity become cryptographically suspect—not due to revocation, but due to plausible forgery enabled by possession of the signing key. This RFC outlines the motivation, design, threat model, and implementation considerations for protocols that wish to support this behavior.

## 2. Background and Motivation

In most cryptographic systems, identity is synonymous with a keypair. The private key represents the capacity to act, and the public key represents the ability to verify. If a private key is lost or stolen, the system treats it as compromised—but it does not typically offer a way to weaponize that compromise in favor of the user.

In practice, however, users often need a way to walk away from their history—not just to stop signing new messages, but to cast doubt on the trustworthiness of past ones. This might be due to personal growth, privacy concerns, safety risks, or social recontextualization. The *burnable identity* offers a new approach: users can **intentionally invalidate the social trust of all previous messages** by making their private key public.

Unlike revocation (which says "this key should no longer be used"), a burn says "this key is now public—anyone can forge what I used to sign."

## 3. Definitions

- **Identity**: A cryptographic persona defined by a root keypair.
- **Root Key**: The primary signing key for the identity, often used to derive additional scoped keys.
- **Burning**: The act of making the root private key public, thereby enabling anyone to forge messages from that identity.
- **Post-Truth State**: A condition in which all messages signed by a now-burned identity are *technically valid* but *socially untrustworthy* due to the key being exposed.

## 4. Threat Model

Burnable identities are not designed to provide forward secrecy or retroactive deletion. Rather, they model social trust dynamics in which:

- An identity may wish to cast doubt on their own history.
- Third parties may encounter signed data and need to determine its *contextual trustworthiness*.
- Systems must allow interpretive trust models (i.e., "I believe this if it was signed *before* the burn, and I trust the burn was real").

The system **does not prevent** bad actors from:
- Forging messages post-burn
- Claiming a burn occurred when it didn’t

Instead, it **embraces ambiguity** as a valid trust layer.

## 5. Burn Mechanism

### 5.1 Key Disclosure

To burn an identity, the user publishes their root private keys inside an [`identity.burn`](./block-types.md) block. Because the keys are revealed, *anyone* can thereafter forge blocks from the identity — which is precisely the point.

### 5.2 Burn Block Format

The `identity.burn` payload (proto `IdentityBurn`, see [Block Types](./block-types.md)) carries:

```json
{
  "identity":            "<identity_address>",
  "revealed_ml_dsa":  "<base64 mldsaKey.private>",
  "revealed_kyber":      "<base64 kyberKey.private>",
  "burn_notice":         "voluntary"
}
```

- `revealed_ml_dsa` / `revealed_kyber` — the identity's root **private** keys, the material that puts the identity into the post-truth state. Both are revealed so the whole identity (authorship *and* prior private-audience confidentiality) is repudiated.
- `burn_notice` — a reason enum: `voluntary` | `compromised` | `rotated` | `other`.
- The block is normally a [plaintext block](./encryption.md) addressed to a [public audience](./audiences.md) so it is world-readable.

### 5.3 Verification

A client treats a burn as **genuine** only if the revealed private keys actually belong to the named identity:

```ts
function VerifyBurn(burn, world):
  d = MLDSAPairFromPrivate(burn.revealed_ml_dsa)
  k = KyberPairFromPrivate(burn.revealed_kyber)
  return BytesToAddress(d.pub, k.pub) == burn.identity
```

This check requires no trusted registry: the revealed keys either re-derive the identity's address (per [Derivations](./derivations.md)) or they do not. A burn whose keys do not match the claimed address is ignored.

> ⚠️ **No anti-replay, by design.** Once the keys are public, anyone can publish a fresh, validly-signed `identity.burn` for that identity at any time. There is no meaningful "first" or "authentic" burn after disclosure — possession of the keys *is* the burn. Clients therefore key their trust decisions off the *fact of disclosure*, not off any single burn block's timestamp.

### 5.4 Client Behavior in the Post-Truth State

Once a client has verified a burn for an identity, it SHOULD:

- Mark **all** blocks authored by that identity — both before and after the burn timestamp — as `contested` / `unverifiable`. Pre- and post-burn blocks are cryptographically indistinguishable from forgeries once the key is public, so a timestamp cutoff offers only heuristic, not cryptographic, assurance.
- Continue to *store and display* such blocks if it wishes (burning is not deletion) but surface their contested status in the UI.
- Optionally apply user-chosen heuristics (e.g. "trust blocks I personally saw mirrored before the burn"), understanding these are social, not cryptographic, judgments.

### 5.5 Revocation vs Poisoning

Traditional revocation is binary and trusted: "This key is revoked via registry."
Burning is subjective and decentralized: "This key is now public. Trust at your own risk."

It is not about deleting data. It's about **creating uncertainty**.

## 6. Signature Interpretation in Post-Truth

Clients and verifiers encountering a signature from a burned identity must decide:
- Do they trust the signature knowing the key is now public?
- Was the signature created *before* or *after* the burn?
- Do they believe the burn was legitimate and intentional?

There is no global truth. Only context. This is the essence of the Post-Truth State.

## 7. Implementation Recommendations

- BlockParty supports identity burn as the first-class [`identity.burn`](./block-types.md) block type (§5.2).
- Clients should maintain local trust models for burned identities, verifying disclosure per §5.3 and applying the post-truth behavior in §5.4.
- Blocks signed by burned keys may remain technically valid, but clients should annotate them as "unverifiable" or "contested."
- Time-based heuristics (e.g., trusting pre-burn blocks but not post-burn ones) may be implemented per user preference, with the caveat in §5.4 that they are social rather than cryptographic.

## 8. Use Cases

- **Social growth**: Users abandon a past version of themselves.
- **Political safety**: Dissidents poison past messages to avoid retroactive persecution.
- **Cultural forgiveness**: Communities normalize walking away from an old identity.
- **Experimental personas**: Temporary identities that are explicitly meant to expire.

## 9. Limitations

- Burn signals are only effective if the system observes them.
- Malicious actors may fake burns to sow confusion.
- Burned content may still be mirrored, cached, or archived.
- There is no cryptographic deletion—only ambiguity.

## 10. References

- [BlockParty Protocol Whitepaper (2025)](https://bpprotocol.org/whitepaper)  
- [PGP Revocation Certificates – GNU Privacy Handbook](https://www.gnupg.org/gph/en/manual/x334.html)  
- [Signal Safety Number Verification Protocol](https://signal.org/docs/specifications/safety-numbers/)  
- [Off-the-Record Messaging (OTR) Protocol v3](https://otr.cypherpunks.ca/Protocol-v3-4.0.0.html)  
- [DID Core Spec – Deactivation & Revocation](https://www.w3.org/TR/did-core/#did-deactivation)

