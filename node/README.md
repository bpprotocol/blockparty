# bpnode — BlockParty node server (Go)

Headless daemon that participates in a **single World's** block-exchange network and exposes a local API for frontend clients (the Electron desktop client, [#26](https://github.com/bpprotocol/blockparty/issues/26)). Tracking epic: **[#25](https://github.com/bpprotocol/blockparty/issues/25)**.

It is an **application built on the Go reference SDK** ([`../implementations/go`](../implementations/go)) — its own Go module (`github.com/bpprotocol/blockparty/node`) so heavy networking/storage deps stay out of the lean SDK. The SDK provides all protocol logic; the node adds networking, persistence, and the client API.

> **Status:** scaffold ([#27](https://github.com/bpprotocol/blockparty/issues/27)) + local storage ([#30](https://github.com/bpprotocol/blockparty/issues/30)) + keystore ([#28](https://github.com/bpprotocol/blockparty/issues/28)) + World guard ([#31](https://github.com/bpprotocol/blockparty/issues/31)). Config, mode selection, the operational API server, graceful lifecycle, the block store/index, the personal-mode keystore, and the ingress validation pipeline are in place. libp2p, exchange, and the full client API land in the remaining #25 sub-issues.

## Modes

The same binary runs in one of two modes (see [#25 → Node roles & key custody](https://github.com/bpprotocol/blockparty/issues/25)):

| Mode | Holds | Can | Runs on |
|------|-------|-----|---------|
| **personal** (default) | keystore: World seed **+** identity keys | author, sign, decrypt, run connections, mirror | the user's own machine |
| **relay** | World **public** key only | validate `world_sig`, store, gossip ciphertext | anywhere, incl. untrusted hosts |

Either mode may boot with **no World loaded** and be configured at runtime by a client (`/statusz` reports `world_loaded`); this is the onboarding flow decided in [#26 Q5](https://github.com/bpprotocol/blockparty/issues/26).

## Run

```sh
go run ./cmd/bpnode --mode relay
go run ./cmd/bpnode --mode personal      # World seed via env (below)
```

```sh
curl http://127.0.0.1:4400/healthz       # {"status":"ok"}
curl http://127.0.0.1:4400/statusz       # mode, world_loaded, fingerprint, uptime, metrics
```

## Configuration

Resolved from (increasing precedence): **defaults → JSON config file → environment → flags**.

| Flag | Env | Default | Notes |
|------|-----|---------|-------|
| `--mode` | `BPNODE_MODE` | `personal` | `personal` \| `relay` |
| `--data-dir` | `BPNODE_DATA_DIR` | OS config dir `/blockparty/node` | |
| `--api-addr` | `BPNODE_API_ADDR` | `127.0.0.1:4400` | loopback by default ([#29](https://github.com/bpprotocol/blockparty/issues/29)) |
| `--log-level` | `BPNODE_LOG_LEVEL` | `info` | `debug`\|`info`\|`warn`\|`error` |
| `--log-format` | `BPNODE_LOG_FORMAT` | `text` | `text`\|`json` |
| `--world-pubkey` | `BPNODE_WORLD_PUBKEY` | — | relay: World ML-DSA-65 public key (hex) |
| `--p2p-listen` | `BPNODE_P2P_LISTEN` | — | reserved ([#32](https://github.com/bpprotocol/blockparty/issues/32)) |
| `--bootstrap` | `BPNODE_BOOTSTRAP` | — | reserved ([#33](https://github.com/bpprotocol/blockparty/issues/33)) |
| `--config` | `BPNODE_CONFIG` | — | path to a JSON config file |
| — | `BPNODE_KEYSTORE_PASSPHRASE` | — | personal: unlocks (or, with a seed, first-time initializes) the encrypted keystore. **Secret** — env only. |
| — | `BPNODE_WORLD_SEED` | — | personal: World seed phrase. **Secret** — env only, never a flag. Used once to initialize the keystore; thereafter the node unlocks with just the passphrase. A relay handed a seed is rejected. |
| — | `BPNODE_IDENTITY_PASSPHRASE` | — | personal: identity passphrase, used when first initializing the keystore. **Secret** — env only. |

## Keystore (personal mode)

Personal mode persists **only secrets** — the World seed phrase and identity passphrase — in `<data-dir>/keystore.json`, encrypted at rest (XChaCha20-Poly1305 under an Argon2id-derived key). **Private keys are never written to disk**; they are re-derived in memory at unlock via the SDK (`OpenWorld` / `OpenIdentity`), exactly reproduced from the same secrets.

```sh
# First run: initialize the keystore from a seed (one time).
BPNODE_KEYSTORE_PASSPHRASE=unlock BPNODE_WORLD_SEED="…seed…" \
  BPNODE_IDENTITY_PASSPHRASE=idpass bpnode --mode personal

# Later runs: unlock with just the passphrase — no seed in the environment.
BPNODE_KEYSTORE_PASSPHRASE=unlock bpnode --mode personal
```

Identity rotation (new identity; the old one stays revocable via `identity.burn`) and burn-material extraction are available on the keystore for the client API (#38) and connections (#37). On a multi-user/remote host an OS-keychain backend (Keychain / libsecret / DPAPI) can slot in behind the same API.

## API framework decision (#27)

The client API is **[Connect](https://connectrpc.com/connect)** (`connectrpc.com/connect`):

- serves over standard `net/http` — the operational endpoints here (`/healthz`, `/statusz`) and the Connect service handlers ([#38](https://github.com/bpprotocol/blockparty/issues/38)) mount on the **same mux**, so this scaffold is forward-compatible with no rework;
- native server-streaming for the subscription/live-feed surface;
- first-class generated **TypeScript** clients for the desktop frontend ([#26](https://github.com/bpprotocol/blockparty/issues/26)), via `@connectrpc/connect-web`.

No protobuf/codegen is pulled in yet — the typed RPC services arrive with #38.

## Layout

```
node/
├── cmd/bpnode/        # daemon entrypoint
└── internal/
    ├── config/        # config: defaults → file → env → flags, + validation
    ├── obs/           # slog logger + lightweight metrics counters
    ├── world/         # which World the node serves (personal vs relay)
    ├── keystore/      # encrypted-at-rest secret store; derives keys in memory (#28)
    ├── store/         # BadgerDB block store + go-memdb index + filesystem blobs (#30)
    ├── guard/         # ingress validation: world_sig gate → dedupe → author → store (#31)
    ├── api/           # net/http server: /healthz, /statusz (Connect handlers land in #38)
    └── app/           # daemon: wiring + Run(ctx) + graceful shutdown
```

## Develop

```sh
go test ./...
go vet ./...
gofmt -l .     # should print nothing
```
