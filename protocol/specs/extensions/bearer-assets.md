---
Title: Bearer Assets Extension
Version: 0.1.0
Last Updated: 2025-04-25
Status: Draft
License: CC0
Canonical URL: https://bpprotocol.org/specs/extensions/bearer-assets
---

# 💰 Bearer Assets Extension

## Overview

BlockParty Bearer Assets provide a lightweight, delay-tolerant mechanism for creating, transferring, and verifying ownership of digital goods without centralized consensus, blockchains, or external infrastructure.

Ownership of a bearer asset is based on possession of cryptographic proof and optional double-spend defenses. Asset lifecycles are self-contained and voluntary, matching BlockParty's broader principles of trust as subjective and association as opt-in.

Bearer assets extend the BlockParty model from speech to value, enabling portable digital ownership in fully decentralized, partially connected, or even offline environments.

## Asset Models

Each asset class specifies at mint time its **Spend Validation Mode**:

| Mode          | Description |
|---------------|-------------|
| `transparent` | Spend proofs must be published to a known audience. Double-spends are detectable by public observation. |
| `issuer`      | Spend proofs must be submitted to the asset's issuing authority. Double-spends are tracked by issuer's nullifier list. |

Participants choose whether to accept assets based on the declared model and their subjective trust in the issuer or audience.

## Block Types

### `asset.mint`
- Issuer identity creates a new asset class and instance.
- Fields:
  - `assetID`: Hash(commitment or secret)
  - `class`: Text label (e.g., "BP Poster 2025")
  - `metadataHash`: (optional) Hash of external metadata (e.g., IPFS link)
  - `validationMode`: `transparent` or `issuer`
  - `validationAudience`: (required if `transparent`) Raw audience seed string
  - `issuerSignature`: Signature by issuer over above fields

### `asset.spend`
- Holder proves ownership and transfers the asset.
- Fields:
  - `assetID`: ID of the asset being spent
  - `nullifier`: Unique derived hash to prevent/reveal double-spends
  - `proof`: ZK proof showing knowledge of asset's secret and optionally chaining from previous spends
  - `recipientPubKey`: (optional) If recipient is known and encryption desired
  - `previousProofHash`: (optional) Hash of previous spend proof if using recursive proofs

### `asset.burn` (optional)
- Holder declares asset void.
- Same structure as spend but includes a "burn" tag.

## Validation Audience Derivation

When using `transparent` mode, the `validationAudience` field in the `asset.mint` block specifies the audience seed.

To derive the world audience code:

1. Prepend the following domain-scoped prefix to the `validationAudience` field:

```
bpprotocol.org/v1/chan/asset.tx/
```

2. Concatenate the prefix and the raw field value.

3. Hash the resulting string according to BlockParty world audience hashing standards.

4. The resulting hash is the canonical audience ID where spend nullifiers must be published.

This mechanism ensures future compatibility, prevents collisions, and enables namespace versioning.

## Custody and Transfer

- **Mint**: Issuer creates and signs the initial asset definition.
- **Possession**: Holder maintains asset proof (and spend secret if applicable).
- **Spend**: To transfer, holder generates a new spend proof and nullifier.
- **Validation**: Receiver verifies proof validity and double-spend status according to asset's spend mode.
- **Lifecycle**: Assets can circulate indefinitely, or optionally be burned.

## Double-Spend Prevention

- In `transparent` mode, double-spends are detected by observing conflicting nullifiers on the declared audience.
- In `issuer` mode, the issuing authority maintains a simple nullifier registry.

Participants are free to accept or reject assets based on their assessment of spend validation confidence.

## Use Cases

### Digital Collectibles
- Portable, delay-tolerant collectibles like posters, badges, or digital artifacts.
- Movable through sneakernet, offline sharing, or low-connectivity worlds.

### Event Tickets / Access Passes
- Issued bearer-style by event organizers.
- Verifiable without relying on centralized ticketing platforms.

### Certifications / Badges
- Proofs of completion, participation, or skill.
- Survivable and verifiable even if the issuing authority later disappears.

### Micro-Economies and World Currencies
- Worlds may issue local currencies or barter tokens based on bearer asset models.
- Fully voluntary economic systems without platform mediation.

### Delay-Tolerant Gifting and Receipts
- Signed proofs of transactions, donations, or personal exchanges.
- Durable even across fragmented or censored networks.

## Design Philosophy

BlockParty Bearer Assets are built to:
- Require no blockchain, fees, or mining.
- Make trust an explicit, subjective choice.
- Embrace delay-tolerant, opportunistic communication.
- Protect ownership through math, not platform policies.
- Empower local cultures and economies without globalized risk.

Bearer assets extend BlockParty's commitment to consent, self-determination, and freedom — not by replacing speech, but by elevating possession itself to the same plane of voluntary, verifiable action.

## Notes

- Proofs may be recursive to maintain constant asset size across many transfers.
- Asset metadata is optional and externalized for flexibility.
- Asset systems are modular; clients are free to support or ignore bearer asset classes.

