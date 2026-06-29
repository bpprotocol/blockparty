# BlockParty desktop client

Cross-platform desktop client for BlockParty: an **Electron** shell wrapping a
**Nuxt 3 SPA** renderer. It is a thin GUI over the local [node server](../../node)
— it does not speak libp2p or manage storage itself; it talks to the node's API.
Tracking epic: **[#26](https://github.com/bpprotocol/blockparty/issues/26)**.

> **Status:** scaffold ([#40](https://github.com/bpprotocol/blockparty/issues/40)) + node API client & health UI ([#41](https://github.com/bpprotocol/blockparty/issues/41)) + node lifecycle ([#42](https://github.com/bpprotocol/blockparty/issues/42)). The app manages (or attaches to) the local node and shows its health. Onboarding (#43), feed (#44), and the rest follow.

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

## Develop

```sh
pnpm install
pnpm dev            # nuxt dev server + electron window (requires a display)
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
│   ├── ipc.ts           # ipcMain handlers (node RPCs + lifecycle)
│   └── gen/node_pb.ts   # generated from node/proto/v1/node.proto
├── app.vue              # renderer root (connection/health UI)
├── types/window.d.ts    # attaches the bridge type to Window
├── nuxt.config.ts       # ssr: false static SPA
├── electron-builder.yml
└── eslint.config.mjs
```
