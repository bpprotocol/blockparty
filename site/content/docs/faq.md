---
title: Frequently Asked Questions
short: FAQ
version: 1.0.0
updated: 2025-05-01
status: Draft
license: CC0
canonical: https://bpprotocol.org/docs/faq
source: https://github.com/bpprotocol/blockparty/blob/master/docs/faq.md
---

# Frequently Asked Questions

What BlockParty Isn’t

Because this project isn't like anything you've seen before.

---

## 🧍 BlockParty: FAQ for Humans

> Start here if you're just trying to understand what BlockParty is, why it matters, and how it feels to use — no technical background required.

---

### ❓ What is BlockParty, in plain English?

**BlockParty is a new kind of social network — but it's not a website or an app.**  
It's a protocol — a set of rules — that lets people post, message, and share without being tracked, censored, or permanently recorded.  
You can think of it like digital pen pals, but safer, private, and fully in your control.

---

### ❓ Is this just another social media platform?

Nope. There’s no company, no server, no feed algorithm.  
**BlockParty is like email or BitTorrent — it’s a protocol that anyone can use, build on, or ignore.**  
It’s not trying to be the next Facebook. It’s trying to make sure you don’t need Facebook.

---

### ❓ Can I use it without knowing anything about crypto?

Yes.  
BlockParty uses **cryptography** behind the scenes to protect your privacy, but you don't need to know how it works.  
Just like you don’t need to understand HTTPS to use a secure website.

---

### ❓ What makes BlockParty different from Signal or Telegram?

Signal and Telegram still rely on **servers** and **companies**.  
BlockParty can work **without a company or a server**.  
You could pass a message on a flash drive. Or over the internet. Or even by scanning a QR code.  
It’s built to work **even if the internet is broken.**

---

### ❓ Why would I want my messages to disappear?

Because sometimes **you grow**.  
People shouldn’t be punished forever for something they said years ago.  
BlockParty lets messages **fade away** unless people choose to keep them.  
That’s more like real life — and way more humane.

---

### ❓ Can I really delete my identity?

Yes. You can “burn” your identity at any time.  
That means even if someone has your old posts, **they can no longer prove you wrote them.**  
It’s called *plausible deniability* — and it’s built into the system on purpose.

---

### ❓ What if someone uses this for something illegal?

Any technology that protects privacy can be abused — like phones, cash, or cars.  
BlockParty is designed for **free speech**, not crime.  
It helps journalists, exiles, whistleblowers, and everyday people have control over their communication.

---

### ❓ Is this like blockchain or Bitcoin?

Not really.  
BlockParty **uses blocks**, but it’s not a blockchain. There’s no mining, no coin, no global ledger.  
Each message is its own signed “block” of data. That’s it.

---

### ❓ Do I need to be technical to use it?

No. If someone builds a simple app on top of it, you could use it like any social app.  
The complex stuff is hidden. You post, comment, reply — just like you always have.  
The difference is: **you control your data.**

---

### ❓ So who controls BlockParty?

No one. And everyone.  
It’s open. You can build on it, fork it, mirror it, or walk away.  
There’s no company to sue, no CEO to ban you, no login screen to lock you out.  
Just people talking — on their own terms.

---


## 🔧 BlockParty: Technicals

> Curious how it actually works? This section dives into the nuts and bolts of BlockParty — from encryption and identity to transport and threading.

---

### ❓ Is this a blockchain?

**No.**  
BlockParty uses *blocks*, but there’s no *chain*.  
Each block is a signed, optionally encrypted unit of communication. It can reference other blocks (threading, reactions, replies), but there’s no global ledger or consensus.

It’s closer to **email with cryptographic structure** than it is to a distributed ledger.

---

### ❓ How is data stored and transmitted?

BlockParty makes **no assumptions about transport**.

Blocks are just binary-encoded protobuf messages. You can transmit them:
- Over TCP, UDP, or mesh
- Via libp2p gossip
- On a USB stick or QR code
- Over sneakernet or satellite

Data is stored however the client/server decides — usually on disk as flat files, indexed by hash, optionally mirrored by others.

---

### ❓ Is the content public or encrypted?

Each block is encrypted to a **specific audience**.

Only recipients with the appropriate key can decrypt it. Everyone else sees only metadata:
- Timestamp
- Block type hash (not human-readable)
- Author/public sigs (if known)
- Audience hash

This makes content **plausibly deniable**, even if observed.

---

### ❓ What cryptographic primitives are used?

The default stack is:
- **Kyber** for post-quantum key exchange and encryption
- **Hash-based audience derivation**
- **Deterministic signatures** (ed25519 or hybrid PQ-safe equivalents)
- All content encryption is audience-scoped — based on shared secrets or derived key pairs

Audiences, types, and channels are all **deterministically derived**, making them portable and predictable without central coordination.

---

### ❓ How does identity work?

Each identity is based on a **root keypair**.

From that root, all keys (signing, encryption, channels, audiences) are derived.  
Publishing a block signed by your key proves authorship.  
Burning your root key (publicly revealing the private key) casts doubt on *all prior blocks*, because anyone could have forged them retroactively.

This creates a **“post-truth state”** — a deliberate, protocol-level feature.

---

### ❓ How is privacy handled between users?

Users exchange encrypted connection requests (think: a two-phase DH exchange).  
Once connected, they receive:
- A private audience channel (derived)
- A private identity block (reveals shared info, keys, etc.)

Users can **rotate their private audiences** for forward secrecy. Friends only see content *from the time of connection forward*, unless access is deliberately granted to older blocks.

---

### ❓ What prevents spam or flooding?

Nothing at the protocol layer — and that’s intentional.  
Spam prevention is a **client-side or transport-layer concern**, not a protocol concern.  
(Think: SMTP vs. Gmail spam filters.)

Possible extensions:
- Optional proof-of-work for public posts
- Mirror filtering based on block type, source, size, etc.
- Trust overlays or web-of-trust mechanisms

---

### ❓ How do threads or replies work?

Each block can include **references to other block IDs** — this enables:
- Comment chains
- Reactions
- Cascading edits
- Thread reconstruction

Because blocks are immutable and individually signed, clients reconstruct threads **locally** from known blocks — no global thread state is required.

---

### ❓ How are large files or media handled?

BlockParty defines two key mechanisms:
- **`chunk.manifest`** – Arbitrary data split into multiple blocks (like torrent pieces)
- **`post.chunked`** – Media-specific manifest with MIME headers + chunked content, enabling streaming or progressive rendering

Clients fetch chunks independently and reconstruct full media on validation.

---

### ❓ What happens if I'm offline? Or the network goes down?

You can:
- Store blocks locally
- Mirror others’ blocks passively
- Export/import blocks via file or device
- Use **no network at all**

This is intentional.  
**BlockParty is post-network capable** — it can function with *no internet*, *no server*, *no time sync*, and still preserve meaning.

---

### ❓ Can I build apps on top of BlockParty?

Yes — that’s the goal.

BlockParty is the **protocol layer**. You can:
- Build clients (chat, forums, social feeds)
- Write bots, schedulers, daemons
- Extend block types
- Define custom RPC methods (e.g. `bp.v1.commerce.PlaceOrder`)
- Use it for scheduling, contracts, marketplaces, identity, or delayed collaboration

You don’t need permission, a login, or an API key.

---

### ❓ Is there a reference implementation?

Yes. It’s open source, written in Go, and modular.  
There’s a basic client-server model (CLI + local HTTP interface), with:
- File-based storage
- Memory and disk-based decryption cache
- Libp2p P2P extension in progress

See: [bpprotocol.org](https://bpprotocol.org) → GitHub

---

## 🧠 BlockParty: Philosophical Questions

> Technology is never neutral. This section explores the deeper ideas behind BlockParty — the why beneath the how. These are the questions that shaped the protocol.

---

### ❓ Why should I care about censorship if I’m not saying anything controversial?

Because **censorship never starts with you** — it starts with those on the edge, and then the edge moves.

You may not care today. But one day, someone **close to your beliefs** or **your past self** might be on the wrong side of an algorithm, a regime, or a social campaign. BlockParty exists **for that person** — and for who you might become.

---

### ❓ What’s so wrong with permanence? Doesn’t it keep people accountable?

Permanence without consent becomes a prison.

We’ve built an internet where **youthful mistakes, half-formed thoughts, and bad jokes are fossilized** — forever searchable, out of context, and judged by future moral standards.  
BlockParty doesn’t erase accountability. It gives people the **possibility of growth** without being chained to their lowest moment.

---

### ❓ Isn’t anonymity dangerous?

**Anonymity is safety.** It protects whistleblowers, survivors, political dissidents, and ordinary people afraid to speak honestly.

Anonymity becomes dangerous **only when used without ethics** — and those ethics aren’t enforced by code. They’re enforced by culture. BlockParty trusts people with power — because **the alternative is to give it to a company or a government.**

---

### ❓ Isn’t decentralization just a technical buzzword?

Not here.  
BlockParty’s decentralization isn’t just about infrastructure — it’s about **dignity**.  
No central server means no central control, no central ban list, no single point of failure for memory, identity, or speech. That’s not a buzzword. That’s *freedom architecture*.

---

### ❓ If a system allows people to disappear, doesn’t that break truth?

BlockParty isn’t built for **truth as a permanent record**.  
It’s built for **intent** — what someone meant, at a moment in time, to a particular audience.

Truth, in BlockParty, is **a living contract** between sender and receiver.  
And like in real life, that contract can end — through distance, disinterest, or mercy.

---

### ❓ Shouldn't we be archiving everything for history?

Not everything deserves to be history.

BlockParty doesn’t prevent memory — it forces memory to be **intentional**.  
If something matters, people can mirror it, store it, preserve it.  
If it doesn’t? Let it fade.  
The internet isn’t sacred. **Meaning is.**

---

### ❓ What if I want to be remembered?

Then people will remember you.

BlockParty doesn’t delete — it simply doesn’t force permanence.  
It lets you **earn your memory** through relationships, care, and relevance.  
That’s a better kind of legacy than a database of tweets.

---

### ❓ Why is the ability to “burn” an identity so important?

Because sometimes the only path forward is to **let a version of yourself die**.

Maybe you were wrong. Maybe you were young. Maybe you were someone else.  
In a digital world where nothing is forgotten, the ability to say “that was me, but it isn’t anymore” — and have that *mean something* — is a radical act of **human respect**.

---

### ❓ What does “post-network” mean?

It means **communication doesn’t have to depend on servers, companies, or live internet**.  
It means **your message still matters**, even if it takes three months to reach someone on Mars.  
It means **humans come before infrastructure** — and our stories deserve to outlive the platforms we use to tell them.

---

### ❓ Is this just about free speech?

It starts there. But it’s also about:
- **Free presence**
- **Free forgetting**
- **Free identity**
- **Free rebuilding**
- **Free relationships that aren’t observed by machines**

It’s about letting people **speak, grow, and change** without needing permission — and without leaving a permanent wound.

---

## Learn More, Go Deeper

The FAQ is just the beginning. Here’s where to go next if you want to understand the architecture, read the philosophy, or get involved.

- Read the [BlockParty Whitepaper](./whitepaper.md) for a full overview of the protocol’s architecture and design principles.  
- Explore the [Foundations Essay Series](/foundations.md) to see how philosophy, culture, and code intertwine.  
- Join the discussion or propose ideas on [GitHub](https://github.com/bpprotocol/blockparty).

**Block by block, this is how we take back communication.**
