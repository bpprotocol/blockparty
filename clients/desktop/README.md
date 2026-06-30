# BlockParty desktop client

Cross-platform desktop client for BlockParty: an **Electron** shell wrapping a
**Nuxt 3 SPA** renderer. It is a thin GUI over the local [node server](../../node)
— it does not speak libp2p or manage storage itself; it talks to the node's API.
Tracking epic: **[#26](https://github.com/bpprotocol/blockparty/issues/26)**.

> **Status:** scaffold ([#40](https://github.com/bpprotocol/blockparty/issues/40)) + node API client & health UI ([#41](https://github.com/bpprotocol/blockparty/issues/41)) + node lifecycle ([#42](https://github.com/bpprotocol/blockparty/issues/42)) + onboarding ([#43](https://github.com/bpprotocol/blockparty/issues/43)) + live feed ([#44](https://github.com/bpprotocol/blockparty/issues/44)) + compose ([#45](https://github.com/bpprotocol/blockparty/issues/45)) + connections ([#46](https://github.com/bpprotocol/blockparty/issues/46)) + identity ([#47](https://github.com/bpprotocol/blockparty/issues/47)). The app manages the node, guides setup, posts, connects to peers, and manages identity keys. Packaging (#48) is the last step.

## Identity & keys (#47)

The dashboard shows the identity card (address, World, signing/KEM public keys) and a **key management** panel for the two dangerous operations:

- **Rotate identity** — set a new passphrase, deriving a new identity (existing connections drop).
- **Burn identity** — publish an `identity.burn` that reveals the root private keys so peers revoke trust. Irreversible; the UI requires typing `BURN` to confirm.

Both go through the node's confirmation gate ([#29](https://github.com/bpprotocol/blockparty/issues/29)): the renderer must pass `confirm: true`, and the node rejects an unconfirmed call (`failed_precondition`). The node holds the keys and performs the operation; no private key ever reaches the renderer. The flow lives in `composables/useIdentity.ts` (unit-tested) and `components/IdentityView.vue`.

## Connections (#46)

A connection is a private, end-to-end channel between two identities. The UI:

- shows **your connection card** (address + public keys) to copy and share out-of-band;
- lets you **paste a peer's card** to register them and start a handshake;
- lists active connections with their epoch, and per-connection **rotate** / **close** and a **private message** thread (decrypted by the node).

The renderer calls the node's connection RPCs (`getIdentity`, `addPeer`, `startConnection`, `listConnections`, `rotate`/`close`, `sendPrivateText`, `listConnectionMessages`) through the bridge; the node runs the actual `connect.*` handshake (#37) and holds the keys. The card encoding (`formatCard`/`parseCard`) and add-and-connect flow live in `composables/useConnections.ts` and are unit-tested.

## Compose (#45)

When the node can author (personal mode), the dashboard shows a compose box: write text, pick a public audience (1–16), and post. The renderer sends only the **intent** (`{ publicAudience, text }`) to the node via `postText` — the node signs (`world_sig` + `author_sig`) and encrypts; the client holds no keys. `content.post` is not a dangerous op, so it needs no confirmation gate (#29). The post then arrives in the live feed (#44) over the block stream. Validation (`validatePost`) lives in `composables/useCompose.ts` and is unit-tested.

## Feed (#44)

The dashboard shows a live feed of the readable posts the node holds:

- **Backlog** via `listBlocks` + `getBlock` (the node decrypts posts on audiences it can open).
- **Live updates** via the node's server-streaming `SubscribeBlocks` RPC. The main process opens the stream (`FeedManager`) and pushes each event to the renderer over IPC (`window.bpDesktop.feed.onEvent`); the renderer fetches the post text and merges it newest-first, de-duplicated by id.

The merge/ordering logic (`mergeItem`) lives in `composables/useFeed.ts` and is unit-tested; the live stream was verified end-to-end against a running node.

## Onboarding (#43)

On connect the app queries the node's status and routes to one of:

| `getStatus`                    | view                                                                           |
| ------------------------------ | ------------------------------------------------------------------------------ |
| disconnected                   | "connecting…" with retry                                                       |
| connected, **no World loaded** | **onboarding** — enter/generate a World seed + identity & keystore passphrases |
| connected, **World loaded**    | dashboard (skips onboarding)                                                   |

Onboarding submits the secrets to the node via `bootstrapWorld` (#38) — the node holds the keys; the renderer never persists them. Once the node reports a loaded World, the view advances to the dashboard. The routing logic (`deriveView`) and the bootstrap flow live in `composables/useNode.ts` and are unit-tested.

## Node lifecycle (#42)

The main process owns the node process via a **supervisor** (`electron/node-supervisor.ts`):

- **Managed mode** (default): spawns the bundled `bpnode`, waits until its API is ready, captures its logs, restarts it with backoff on a crash, and stops it cleanly (SIGTERM → SIGKILL) when the app quits.
- **Attach mode** (`BPNODE_ATTACH=1`): skips spawning and connects to an externally-run daemon at a given endpoint.

The node API token and endpoint are resolved by the supervisor and handed to the API client in the main process — never to the renderer. Lifecycle state (mode, running/crashed, restarts, endpoint) is surfaced to the renderer via `window.bpDesktop.lifecycle`.

Knobs (env-overridable for development):

| Env               | Default                     | Purpose                             |
| ----------------- | --------------------------- | ----------------------------------- |
| `BPNODE_ATTACH`   | —                           | `1` → attach mode (don't spawn)     |
| `BPNODE_BIN`      | `<resources>/bpnode`        | path to the bpnode binary (managed) |
| `BPNODE_API_ADDR` | `127.0.0.1:4400`            | node API address                    |
| `BPNODE_DATA_DIR` | `<appData>/blockparty/node` | node data dir (holds `api.token`)   |
| `BPNODE_MODE`     | `personal`                  | node mode                           |

The packaged app bundles the per-platform `bpnode` into its resources dir (built by the packaging step, #48).

## Node API client (#41)

The renderer never talks to the node directly. The **main process** owns the [Connect](https://connectrpc.com/) client — it holds the bearer token ([#29](https://github.com/bpprotocol/blockparty/issues/29)) and talks to the node over HTTP — and exposes a typed surface to the renderer over IPC:

```
renderer  ──IPC──▶  main (NodeClient + token)  ──HTTP/Connect──▶  bpnode
window.bpDesktop.node.getStatus()
```

- **`electron/gen/node_pb.ts`** — generated from [`node/proto/v1/node.proto`](../../node/proto/v1/node.proto) with `protoc-gen-es` (regenerate: `pnpm gen:proto`). The same proto the node serves, so the client is always in sync.
- **`electron/node-client.ts`** — wraps the generated Connect client, attaches the token, and converts wire messages to plain DTOs (no protobuf types or bigints cross IPC). Errors become `{ ok: false }` so the UI renders connection problems instead of throwing.
- **`electron/bridge.ts`** — the single typed contract (`NodeApi`, DTOs) shared by main, preload, and renderer.
- **`electron/node-config.ts`** — locates the node API (`BPNODE_API_ADDR`, default `127.0.0.1:4400`) and reads its token from the node data dir (`BPNODE_DATA_DIR`). #42 provides these when it manages the bundled `bpnode`.

The renderer's connection/health panel polls `getStatus` and shows node version, mode, World, identity, and block count (or a disconnected state with retry).

## Architecture

- **`electron/main.ts`** — the Electron main process: window lifecycle and (later) managing the bundled `bpnode`. It loads the Nuxt dev server in development and the generated static SPA in production.
- **`electron/preload.ts`** — the **only** bridge between renderer and main. It exposes a small, typed `window.bpDesktop` API via `contextBridge`; the renderer never touches Node, `fs`, or the main process directly.
- **`app.vue` / Nuxt** — the renderer (SPA, `ssr: false`).

### Secure defaults (#40)

The `BrowserWindow` is created with `contextIsolation: true`, `nodeIntegration: false`, `sandbox: true`, and `webSecurity: true`; external links open in the user's browser, not in-app. All privileged capability is added behind the typed preload bridge.

## Run it locally

The desktop app is a thin GUI over a [node server](../../node) — it always needs a node to talk to. In development the app is **not packaged**, so there's no bundled `bpnode` in `process.resourcesPath` for the supervisor (#42) to spawn. Local runs therefore use one of two paths: **attach** to a node you run yourself (simplest), or point the supervisor at a `bpnode` binary you built (`BPNODE_BIN`).

**Prerequisites:** Node ≥ 20 + [pnpm](https://pnpm.io); a **display** (`pnpm dev` opens an Electron window — it can't run headless); and the **Go toolchain** to run/build the node.

### Attach mode (recommended)

Run a node in one terminal and the app — pointed at it — in another:

```sh
# Terminal A · a local node (from the repo root)
cd node
go run ./cmd/bpnode --mode personal        # API on 127.0.0.1:4400, writes <data-dir>/api.token
```

```sh
# Terminal B · the app, attached to that node
cd clients/desktop
pnpm install
BPNODE_ATTACH=1 pnpm dev                    # don't spawn a binary; connect to the running node
```

`BPNODE_ATTACH=1` tells the supervisor to **skip spawning** and connect to the node at `BPNODE_API_ADDR` (default `127.0.0.1:4400`), reading its bearer token from `BPNODE_DATA_DIR`. If your node runs on a non-default address or data dir, set `BPNODE_API_ADDR` / `BPNODE_DATA_DIR` on the app to match. On first launch the app's **onboarding** (#43) walks you through creating/entering a World; once the node reports a World loaded, it advances to the dashboard. See the [node README](../../node/README.md) for keystore/World env (`BPNODE_KEYSTORE_PASSPHRASE`, `BPNODE_WORLD_SEED`, …).

### Managed mode (the app spawns the node)

Build the node once and let the app supervise it — exercising the managed lifecycle (#42) the packaged app uses:

```sh
cd node && go build -o /tmp/bpnode ./cmd/bpnode
cd ../clients/desktop
BPNODE_BIN=/tmp/bpnode pnpm dev             # spawn, supervise, and stop the node with the app
```

All lifecycle knobs (`BPNODE_ATTACH`, `BPNODE_BIN`, `BPNODE_API_ADDR`, `BPNODE_DATA_DIR`, `BPNODE_MODE`) are documented in [Node lifecycle (#42)](#node-lifecycle-42).

## Develop

```sh
pnpm install
pnpm dev            # nuxt dev server + electron window (requires a display + a node; see "Run it locally")
```

```sh
pnpm build:renderer # nuxt generate → .output/public (static SPA)
pnpm build:main     # tsc → dist-electron (main + preload, CommonJS)
pnpm build          # both, then electron-builder → release/ (mac/win/linux)
pnpm typecheck      # vue-tsc (renderer) + tsc (electron)
pnpm lint           # eslint (flat config: js + ts + vue)
pnpm format         # prettier
```

## Layout

```
clients/desktop/
├── electron/
│   ├── main.ts          # window lifecycle + wires the node IPC
│   ├── preload.ts       # the typed window.bpDesktop bridge
│   ├── bridge.ts        # renderer↔main contract (NodeApi + DTOs)
│   ├── node-client.ts   # Connect client over HTTP (holds the token)
│   ├── node-supervisor.ts # spawn/supervise bpnode, or attach (#42)
│   ├── node-config.ts   # resolve managed/attach options
│   ├── feed-manager.ts  # owns the live SubscribeBlocks stream → renderer (#44)
│   ├── ipc.ts           # ipcMain handlers (node RPCs + lifecycle + feed)
│   └── gen/node_pb.ts   # generated from node/proto/v1/node.proto
├── app.vue              # renderer root (routes connecting/onboarding/dashboard)
├── composables/         # useNode, useFeed, useCompose, useConnections, useIdentity
├── components/          # Onboarding, NodeDashboard, Compose, Connections, Identity, Feed
├── types/window.d.ts    # attaches the bridge type to Window
├── nuxt.config.ts       # ssr: false static SPA
├── electron-builder.yml
└── eslint.config.mjs
```
