---
Title: Direct Message Block Type
Version: 0.1.0
Last Updated: 2025-05-01
Status: Draft
License: CC0
Canonical URL: https://bpprotocol.org/specs/extensions/dm
---

# `dm.message` – Direct Message Block

Defines a lightweight, audience-scoped message block designed for one-to-one or group conversations. It enables directed, conversational messaging between peers without conflating it with shared posts.

---

## Type Identifier
`bpprotocol.org/v1/types/dm.message`

---

## Purpose

Unlike `content.post`, which behaves like a private or public bulletin post, `dm.message` is a direct, conversational message intended for messaging clients and dialogue-centric experiences.

This block type allows:
- Message threading
- Explicit conversational intent
- Lightweight reply structure

---

## Schema
```json
{
  "thread_id": "abc123def456",
  "reply_to": "block_id_of_previous_message",
  "content": {
    "headers": {
      "bpprotocol.org/v1/content-type": "text/plain"
    },
    "body": "Hey, are you around later?"
  }
}
```

### Fields:
- `thread_id`: Required. A unique thread/session ID shared among messages in the same DM thread.
- `reply_to`: Optional. The `block_id` of the message being replied to.
- `content`: Required. Contains MIME-style headers and body, following the `content.post` convention.

---

## Audience Scope

DMs are encrypted and signed like all BlockParty blocks, but their audience is typically:
- One other identity (1:1 chat)
- A small, explicit set (group thread)

Messages may be retained, mirrored, or expire based on client preferences. Clients may show ephemeral threads or durable histories depending on audience intent.

---

## Why No `dm.typing`, `dm.read`, or `dm.reaction`?

Typing indicators and similar UX signals are better implemented via negotiated ephemeral sessions through `client.rallypoint` + application-layer protocols.

Including such features at the block level introduces unnecessary complexity, latency, and overhead for signals that are:
- Transient
- Unreliable over latency-tolerant protocols
- Not semantically durable

---

## Example Use Case

Alice and Bob are connected. Alice sends a `dm.message` block to Bob’s audience channel:
```json
{
  "thread_id": "chat-alice-bob",
  "content": {
    "headers": {
      "bpprotocol.org/v1/content-type": "text/plain"
    },
    "body": "Are you joining the stream?"
  }
}
```
Bob replies with another `dm.message`, referencing the first message by `block_id`.

---

## Notes

- Timestamps are handled at the block level (`ts` field)
- Thread IDs may be deterministically generated from the participants or randomly assigned
- Clients may index DMs separately from content posts

This type serves as a minimal but expressive foundation for conversational communication in BlockParty.

