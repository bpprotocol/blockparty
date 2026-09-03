---
title: Chainlets Extension
version: 0.1.0
updated: 2025-04-26
status: Draft
license: CC0
canonical: https://bpprotocol.org/specs/extensions/chainlets
---

# 📏 Chainlets Extension

## Overview

Chainlets are an optional BlockParty extension that introduce minimal, delay-tolerant, audience-scoped shared state histories between participants.

Chainlets model shared state as a directed acyclic graph (DAG) of signed blocks, similar in structure to Git rather than traditional blockchains. They enable collaborative games, documents, ledgers, agreements, and stateful multi-party interactions without imposing any global consensus or platform dependency.

Chainlets are voluntary, modular, and interpreted by applications — not enforced by the protocol.

## Core Concepts

| Element            | Purpose |
|--------------------|---------|
| `chainlet.genesis`  | Initializes a new chainlet, defining policies and participant expectations. |
| `chainlet.block`    | Appends a new signed state transition or payload to the chainlet. |
| Parent References   | Allow for forks, merges, and voluntary resolution. |
| Confirmation/ Rejection RPCs | Social consensus among participants via signed confirmations or objections. |

## Block Types

### `chainlet.genesis`
- Defines the creation of a new chainlet.
- Fields:
  - `chainletID`: Unique identifier (e.g., hash of genesis block fields)
  - `participants`: (optional) Allowed participant public keys (for closed chainlets)
  - `policies`: (optional) JSON-encoded advisory rules (e.g., minimum confirmations, fork handling preferences)
  - `initialState`: (optional) Starting document, board state, etc.
  - `signature`: Signature by the creator

### `chainlet.block`
- Extends a chainlet by proposing a new state change.
- Fields:
  - `chainletID`: ID of the chainlet being extended
  - `parentHashes`: List of one or more parent block hashes (allowing forks/merges)
  - `payload`: Application-defined state delta or snapshot
  - `signature`: Signature by the author

## Confirmation and Rejection RPCs

Chainlets introduce optional extension RPC methods for participants to signal agreement or objection to proposed blocks.

### `chainlet.confirm`
Participants can confirm blocks they accept by sending signed confirmations.

Fields:
- `chainletID`
- `blockHash`
- `confirmationSignature`
- `metadata` (optional)

### `chainlet.reject`
Participants can reject blocks they object to by sending signed rejections.

Fields:
- `chainletID`
- `blockHash`
- `rejectionSignature`
- `reasonCode` (optional): Short code like "INVALID_MOVE", "FORK_REJECTION"
- `metadata` (optional)

Confirmations and rejections are used voluntarily to:
- Signal agreement or disagreement.
- Help manage forks.
- Enable social resolution mechanisms.

Confirmation and rejection policies are application-layer concerns and are not enforced by the protocol.

## Fork Handling

Chainlets **allow forks** naturally.
- Competing `chainlet.block` entries may exist referencing the same parent.
- Forks are resolved voluntarily by:
  - Application logic
  - Participant consensus via confirmations and rejections
  - World-specific fork resolution policies (optional)

Forks are not catastrophic — they are visible and manageable.

## Application Layer Responsibilities

Chainlet blocks provide cryptographic ordering of actions.
**Applications interpret the meaning of those actions.**

Applications are responsible for:
- Validating whether payloads are legal (e.g., enforcing chess rules, document edit formats, contract constraints).
- Reconstructing full state based on chainlet DAG traversal.
- Resolving forks according to domain-specific rules.
- Defining whether payloads are full state snapshots or incremental deltas.

### State Representation
- **Snapshot Model**: Each block contains the full system state at that point. Easier syncing; larger payloads.
- **Delta Model**: Each block contains only changes from its parent. Smaller payloads; requires replaying history.

Applications are free to choose their state model based on needs.

## Use Cases

### Collaborative Documents
- Participants co-author and proof documents.
- Deltas or full documents stored per block.

### Turn-Based Games
- Sequential moves stored as blocks.
- Game engine validates move legality.

### Voting Systems
- Ballots cast as blocks.
- Tallying rules defined by application/world.

### Multi-party Agreements
- Contract clauses appended and co-signed.
- Forks represent competing interpretations or amendments.

### Shared Ledgers
- Resource exchanges, IOUs, and micro-transactions recorded.

## Design Philosophy

Chainlets extend BlockParty's foundational principles:
- **Consent over consensus**: Participation is voluntary.
- **Freedom over finality**: Forks are normal; social resolution is honored.
- **Speech first, state second**: The protocol transports intent; applications define meaning.
- **Delay-tolerant by nature**: Chainlet evolution does not require constant connectivity.

Chainlets provide a substrate for building worlds of collaboration, trade, competition, and governance — all without sacrificing the sovereign agency BlockParty protects.

## Notes

- Chainlets are scoped to worlds/audiences.
- Confirmation and rejection tracking is optional and advisory.
- Applications are encouraged to publish advisory policies in genesis blocks but cannot enforce them at the transport layer.
- Storage pruning, snapshotting, and other optimizations are left to client/world design.
- Chainlets are not "smart contracts" and are not enforced at protocol level — they are voluntary, verifiable shared histories.

