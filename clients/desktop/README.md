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

The dashboard shows a live feed of **public-audience** posts the node holds:

- **Backlog** via `listBlocks` + `getBlock` (the node decrypts posts on audiences it can open).
- **Live updates** via the node's server-streaming `SubscribeBlocks` RPC. The main process opens the stream (`FeedManager`) and pushes each event to the renderer over IPC (`window.bpDesktop.feed.onEvent`); the renderer fetches the post text and merges it newest-first, de-duplicated by id.

The feed is **scoped to public audiences**: every `BlockSummary` carries a `public_audience` number (`public-N`, or `0` for private/inbox audiences), computed by the node — the only party that can map an `audience_code` back to a well-known public audience. The renderer drops non-public blocks (private connection messages live in the connections UI, #46) and labels each post with its `public-N`.

The merge/ordering (`mergeItem`) and public-only scoping live in `composables/useFeed.ts` and are unit-tested; the live stream was verified end-to-end against a running node.

## Onboarding & unlock (#43)

On connect the app queries the node's status and routes to one of:

| `getStatus`                               | view                                                                           |
| ----------------------------------------- | ------------------------------------------------------------------------------ |
| disconnected                              | "connecting…" with retry                                                       |
| connected, no World, **keystore on disk** | **unlock** — enter the keystore passphrase                                     |
| connected, no World, **no keystore**      | **onboarding** — enter/generate a World seed + identity & keystore passphrases |
| connected, **World loaded**               | dashboard (skips both)                                                         |

**Unlock** covers the node started without `BPNODE_KEYSTORE_PASSPHRASE`: it holds an encrypted keystore it cannot open, so it boots with no World. Status reports `keystoreExists`, the app asks for the passphrase and calls `unlockKeystore`, and the node re-derives the World and identity in memory. Offering onboarding there would fail — the node never overwrites an existing keystore. A wrong passphrase is surfaced in place and the node stays unconfigured.

The unlock screen also carries the way out of a **lost** passphrase: _Forgotten your passphrase?_ reveals a warning and requires typing `CLEAR` before calling `clearKeystore`, which passes the node's `confirm: true` gate (#29) — the same two-step pattern as the identity burn. The node deletes the keystore, status flips to no keystore, and the view falls through to onboarding for a new World.

Onboarding submits the secrets to the node via `bootstrapWorld` (#38) — the node holds the keys; the renderer never persists them. Once the node reports a loaded World, the view advances to the dashboard. The routing logic (`deriveView`) and the bootstrap flow live in `composables/useNode.ts` and are unit-tested.

## Packaging (#48)

`pnpm build` produces an installable artifact for the platform you run it on, with a `bpnode` built for that same target inside it:

```sh
pnpm build       # installer/artifact → release/   (AppImage · dmg · NSIS installer)
pnpm run pack:dir # unpacked app → release/linux-unpacked (etc.), for a quick look
pnpm build:node  # just the bundled node → resources/, for this machine
```

The node is compiled by an electron-builder **`beforePack` hook** (`scripts/before-pack.mjs` → `scripts/build-node.mjs`), which maps the target electron-builder is about to package to a Go target — `darwin/win32/linux` → `GOOS`, `x64/arm64/armv7l/ia32` → `GOARCH` — empties `resources/` and builds `bpnode` (`bpnode.exe` on Windows, the same name `node-config.ts` probes for) with `CGO_ENABLED=0 -trimpath`. Every artifact therefore carries exactly one node, built for itself; `resources/` is generated, never committed.

| Target  | Artifact                                                                  | Built on |
| ------- | ------------------------------------------------------------------------- | -------- |
| Linux   | `BlockParty-<version>-amd64.deb` · `BlockParty-<version>-x86_64.AppImage` | Linux    |
| macOS   | `BlockParty-<version>-<arch>.dmg` (x64, arm64)                            | macOS    |
| Windows | `BlockParty-<version>-x64.exe` (NSIS)                                     | Windows  |

The Go side cross-compiles freely, but each platform's _installer_ has to be produced on that platform (electron-builder needs macOS for a dmg, Windows or wine for NSIS) — so releases come from a matrix, not one machine. The app icon is generated from `build/icon.png`; the executable is named `blockparty` (`executableName`), not after the npm package.

**Code signing, notarization and auto-update are deliberately out of scope** (`mac.identity: null`). The artifacts install and launch unsigned, with the usual first-run warnings on macOS and Windows.

### Linux sandbox (Ubuntu 23.10+)

Chromium refuses to start without a sandbox, and the packaged app keeps `sandbox: true` (#40). On distros with `kernel.apparmor_restrict_unprivileged_userns=1` — Ubuntu 23.10 and later — an unconfined program that creates a user namespace is transitioned into the restrictive `unprivileged_userns` profile, so the **namespace sandbox** is unavailable unless the app has an AppArmor profile of its own. The **SUID sandbox** is the fallback, and it needs `chrome-sandbox` owned by root with mode 4755.

- **`.deb` (recommended on Linux):** handled for you. It installs `/etc/apparmor.d/blockparty` — the same one-line `userns,` profile Chrome, Element and other packaged Electron apps ship — and, on systems with no user namespaces at all, sets `chrome-sandbox` setuid instead.
- **AppImage:** it cannot fix itself. An AppImage mounts `nosuid`, so its `chrome-sandbox` can never be setuid, and its mount path changes every run. Install the profile once:

  ```sh
  sudo install -m644 packaging/apparmor/blockparty-appimage /etc/apparmor.d/
  sudo apparmor_parser -r /etc/apparmor.d/blockparty-appimage
  ```

  (Or, system-wide and much blunter, `sudo sysctl -w kernel.apparmor_restrict_unprivileged_userns=0`.)

Without one of those the app aborts with _"The SUID sandbox helper binary was found, but is not configured correctly"_ or _"No usable sandbox!"_. We deliberately don't ship `--no-sandbox` in the packaged app — that would trade the renderer's isolation for convenience.

First run needs nothing prepared: the app spawns the bundled node against `<appData>/blockparty/node`, the node creates that directory and its `api.token`, and the renderer's onboarding (#43) sets up the World.

> Note: `pnpm pack` is pnpm's own tarball command — the packaging scripts are `pack:dir` / `pack:dir:client-only` so they always reach electron-builder.

## Client-only build & external nodes (#51)

The app can be packaged **without** a bundled `bpnode` — a lighter build for people who already run a node (on this machine, a server, a relay) and just want the UI:

```sh
pnpm build:client-only   # electron-builder --config electron-builder.client-only.yml
pnpm pack:dir:client-only # unpacked, for a quick look
```

What decides the behaviour is the **binary itself, not an env var**: on startup `resolveSupervisorOptions` probes `<resources>/bpnode` (`bpnode.exe` on Windows). Present → managed mode as before. Absent → the app is **attach-only** and never tries to spawn; instead of a spawn `ENOENT` it reports a typed `NodeUnavailableError` and the renderer shows a **"Connect to a node"** screen asking for the node's API address plus either its API token or the data directory to read `api.token` from.

That endpoint is probed before it is saved — an unreachable address or a bad token is reported in the form, not left as a dead app — and then persisted to `<userData>/external-node.json` (mode `0600`; the token stays in the main process, as the managed node's does). Lifecycle state carries `canManage` and `needsEndpoint` so the UI can tell "no node configured yet" from "configured but unreachable"; the latter offers **Use a different node**, and the dashboard's node widget offers **Change node**, both re-opening the same form without a restart.

Env overrides still win for development: `BPNODE_ATTACH=1` attaches to `127.0.0.1:4400` (or `BPNODE_API_ADDR`), and `BPNODE_BIN` points managed mode at a node you built yourself.

## Node lifecycle (#42)

The main process owns the node process via a **supervisor** (`electron/node-supervisor.ts`):

- **Managed mode** (default _when a `bpnode` ships with the build_): spawns the bundled `bpnode`, waits until its API is ready, captures its logs, restarts it with backoff on a crash, and stops it cleanly (SIGTERM → SIGKILL) when the app quits.
- **Attach mode** (`BPNODE_ATTACH=1`, or automatically in a client-only build): skips spawning and connects to an externally-run daemon at a given endpoint.

The node API token and endpoint are resolved by the supervisor and handed to the API client in the main process — never to the renderer. Lifecycle state (mode, running/crashed, restarts, endpoint) is surfaced to the renderer via `window.bpDesktop.lifecycle`.

Knobs (env-overridable for development):

| Env               | Default                     | Purpose                             |
| ----------------- | --------------------------- | ----------------------------------- |
| `BPNODE_ATTACH`   | —                           | `1` → attach mode (don't spawn)     |
| `BPNODE_BIN`      | `<resources>/bpnode`        | path to the bpnode binary (managed) |
| `BPNODE_API_ADDR` | `127.0.0.1:4400`            | node API address                    |
| `BPNODE_DATA_DIR` | `<appData>/blockparty/node` | node data dir (holds `api.token`)   |
| `BPNODE_MODE`     | `personal`                  | node mode                           |

A build with no bundled binary resolves to attach-only on its own (#51); `BPNODE_ATTACH` is only needed to force it when a binary _is_ present.

The packaged app bundles the per-platform `bpnode` into its resources dir — see [Packaging (#48)](#packaging-48).

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

- **`electron/main.ts`** — the Electron main process: window lifecycle and managing the bundled `bpnode`. It loads the Nuxt dev server in development, and in production serves the generated SPA over a privileged `app://bundle` scheme. That scheme is not cosmetic: Nuxt emits absolute asset paths (`/_nuxt/…`), which `file://` resolves against the filesystem root, and its entry is an ES module, which browsers refuse to load cross-origin from a file URL — either one leaves a blank window. The handler resolves paths inside `.output/public` and refuses anything outside it.
- **`electron/preload.ts`** — the **only** bridge between renderer and main. It exposes a small, typed `window.bpDesktop` API via `contextBridge`; the renderer never touches Node, `fs`, or the main process directly.
- **`app.vue` / Nuxt** — the renderer (SPA, `ssr: false`).

### Window chrome

The window is **frameless**: there is no system title bar, and the app's own header is the drag region (`-webkit-app-region: drag`, with buttons, chips and avatars opted back out so they stay clickable). Double-clicking it maximizes and restores, as a title bar would. On Windows and Linux the header carries the app's minimize / maximize / close buttons; macOS uses `titleBarStyle: 'hiddenInset'` instead, keeping its native traffic lights, and the header reserves space for them. The login/setup screen has no header, so it gets a slim drag strip with the same buttons over the photo.

Window state is driven from the renderer over the bridge (`window.bpDesktop.window`), and the main process pushes maximize/restore back, so the button matches the window even when the window manager changes it. `useWindow` decides per platform what to draw; both rules are unit-tested.

Electron's default File/Edit/View/Window/Help menu is removed — the app's own left rail is its navigation. macOS keeps a minimal menu built from the standard roles, because the system always shows a menu bar for the focused app and `Cmd+Q`, `Cmd+C`/`Cmd+V` and `Cmd+W` come from it. `autoHideMenuBar` stops Alt from summoning a bar on Windows/Linux. Losing the View menu also loses its DevTools shortcut, so development builds bind `F12` / `Ctrl+Shift+I` (`Cmd+Alt+I`) directly; packaged builds do not.

### Secure defaults (#40)

The `BrowserWindow` is created with `contextIsolation: true`, `nodeIntegration: false`, `sandbox: true`, and `webSecurity: true`; external links open in the user's browser, not in-app. All privileged capability is added behind the typed preload bridge.

## Run it locally

The desktop app is a thin GUI over a [node server](../../node) — it always needs a node to talk to. In development the app is **not packaged**, so there's no bundled `bpnode` in `process.resourcesPath` for the supervisor (#42) to spawn. Local runs therefore use one of two paths: **attach** to a node you run yourself (simplest), or point the supervisor at a `bpnode` binary you built (`BPNODE_BIN`).

**Prerequisites:** Node ≥ 20 + [pnpm](https://pnpm.io); a **display** (`pnpm dev` opens an Electron window — it can't run headless); and the **Go toolchain** to run/build the node.

> **First install:** pnpm blocks dependency build scripts by default, but `electron`, `esbuild`, and `@parcel/watcher` are approved in `pnpm-workspace.yaml`, so `pnpm install` downloads Electron's platform binary automatically. If you ever see _"Electron failed to install correctly"_, run `pnpm rebuild electron` (or `rm -rf node_modules && pnpm install`).

> **Linux sandbox:** on kernels that restrict unprivileged user namespaces (e.g. Ubuntu ≥ 23.10 / 24.04), Chromium's SUID sandbox aborts with _"chrome-sandbox … owned by root … mode 4755."_ The `dev` script passes `--no-sandbox` to work around this **in development only** — the packaged app keeps `sandbox: true` (#40) and handles it properly; see [Linux sandbox (Ubuntu 23.10+)](#linux-sandbox-ubuntu-2310). To instead keep the dev sandbox, make the helper setuid root (`sudo chown root node_modules/.pnpm/electron@*/node_modules/electron/dist/chrome-sandbox && sudo chmod 4755 …`) or relax the sysctl.

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
pnpm build:node     # go build → resources/bpnode for this machine (#48)
pnpm build          # renderer + main + a target-matched bpnode, then electron-builder → release/
pnpm build:client-only # same, without a bundled bpnode → release-client-only/ (#51)
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
│   ├── node-config.ts   # detect a bundled node; resolve managed/attach options (#42, #51)
│   ├── external-node.ts # the saved external endpoint + the connect flow (#51)
│   ├── feed-manager.ts  # owns the live SubscribeBlocks stream → renderer (#44)
│   ├── ipc.ts           # ipcMain handlers (node RPCs + lifecycle + feed)
│   └── gen/node_pb.ts   # generated from node/proto/v1/node.proto
├── app.vue              # renderer root (routes connecting/onboarding/dashboard)
├── composables/         # useNode, useFeed, useCompose, useConnections, useIdentity
├── components/          # Onboarding, NodeDashboard, Compose, Connections, Identity, Feed
├── types/window.d.ts    # attaches the bridge type to Window
├── nuxt.config.ts       # ssr: false static SPA
├── scripts/
│   ├── build-node.mjs   # build bpnode for a packaging target (#48)
│   ├── before-pack.mjs  # electron-builder hook that calls it
│   └── deb-after-*.sh   # deb postinst/postrm: AppArmor profile + sandbox setup
├── build/icon.png       # app icon, rendered into per-platform icons
├── build/linux/apparmor-profile     # installed by the deb
├── packaging/apparmor/blockparty-appimage # the same, for AppImage users
├── electron-builder.yml
├── electron-builder.client-only.yml  # the no-bundled-node build target (#51)
└── eslint.config.mjs
```
