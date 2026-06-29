# BlockParty desktop client

Cross-platform desktop client for BlockParty: an **Electron** shell wrapping a
**Nuxt 3 SPA** renderer. It is a thin GUI over the local [node server](../../node)
— it does not speak libp2p or manage storage itself; it talks to the node's API.
Tracking epic: **[#26](https://github.com/bpprotocol/blockparty/issues/26)**.

> **Status:** scaffold ([#40](https://github.com/bpprotocol/blockparty/issues/40)) + node API client & connection/health UI ([#41](https://github.com/bpprotocol/blockparty/issues/41)). The app talks to a running node and shows its health. Node lifecycle (#42), onboarding (#43), feed (#44), and the rest follow.

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
│   ├── node-config.ts   # locate node API + read its token
│   ├── ipc.ts           # ipcMain handlers per RPC
│   └── gen/node_pb.ts   # generated from node/proto/v1/node.proto
├── app.vue              # renderer root (connection/health UI)
├── types/window.d.ts    # attaches the bridge type to Window
├── nuxt.config.ts       # ssr: false static SPA
├── electron-builder.yml
└── eslint.config.mjs
```
