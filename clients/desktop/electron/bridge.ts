// The contract between the renderer and the main process, exposed on
// window.bpDesktop via the preload bridge. Plain DTOs (no protobuf types or
// bigints) cross the IPC boundary, so the renderer stays decoupled from the
// node's wire format.

// Result is a discriminated envelope: RPCs never throw across IPC; a transport
// or node error becomes { ok: false }, which the UI renders as a connection
// problem.
export type Result<T> = { ok: true; value: T } | { ok: false; error: string }

export interface NodeStatus {
  version: string
  mode: string
  worldLoaded: boolean
  world: string
  identity: string
  blockCount: number
  canAuthor: boolean
  // An encrypted keystore is on disk. With worldLoaded false the client must
  // offer unlock rather than onboarding — a keystore is never overwritten.
  keystoreExists: boolean
}

export interface BootstrapRequest {
  worldSeed: string
  identityPassphrase: string
  keystorePassphrase: string
}

export interface BootstrapResult {
  world: string
  identity: string
}

export interface PostTextRequest {
  publicAudience: number
  text: string
}

export interface BlockSummary {
  id: string
  audienceCode: string
  typeCode: string
  author: string
  timestamp: number
  receivedAt: number
  publicAudience: number // public-N (1..16) if a well-known public audience, else 0 (#44)
}

export interface BlockView {
  summary: BlockSummary
  decrypted: boolean
  text: string
}

export interface ListFilter {
  audienceCode?: string
  typeCode?: string
  author?: string
  from?: number
  to?: number
}

// NodeApi is the typed node surface the renderer may call. Implemented in the
// main process (which holds the bearer token and talks to the node over HTTP)
// and bridged to the renderer over IPC.
export interface NodeApi {
  getStatus(): Promise<Result<NodeStatus>>
  bootstrapWorld(req: BootstrapRequest): Promise<Result<BootstrapResult>>
  // unlockKeystore opens a keystore the node already holds, using the
  // passphrase it was created with. The node derives the keys in memory; the
  // passphrase is not persisted by the client.
  unlockKeystore(keystorePassphrase: string): Promise<Result<BootstrapResult>>
  // clearKeystore deletes the node's keystore, discarding the World seed and
  // identity it protects — the recovery path for a lost passphrase. confirm
  // must be the user's explicit confirmation; the node rejects it otherwise.
  clearKeystore(confirm: boolean): Promise<Result<void>>
  postText(req: PostTextRequest): Promise<Result<{ id: string }>>
  getBlock(id: string): Promise<Result<BlockView>>
  listBlocks(filter: ListFilter): Promise<Result<BlockSummary[]>>
}

// --- Node lifecycle (#42) ---

export type LifecycleMode = 'managed' | 'attach'
export type NodeState = 'starting' | 'running' | 'crashed' | 'stopped'

export interface LifecycleState {
  mode: LifecycleMode
  state: NodeState
  endpoint: string // '' until an attach-only app has an endpoint
  restarts: number
  // canManage: this build ships a bpnode it can run itself. False in the
  // client-only package (#51) — the app can only attach to an external node.
  canManage: boolean
  // needsEndpoint: attach-only with no node configured yet, so the UI must ask
  // for one rather than report a connection failure.
  needsEndpoint: boolean
}

export interface LifecycleApi {
  getState(): Promise<LifecycleState>
  recentLogs(): Promise<string[]>
}

// --- External node (#51) ---

// ExternalNodeConfig points the app at a node someone else runs: its API base
// URL plus a bearer token, either pasted or read from the node's data dir.
export interface ExternalNodeConfig {
  baseUrl: string
  token?: string
  dataDir?: string
}

// ExternalNodeApi lets the renderer read and set that endpoint. set() probes
// the node before saving, so a bad URL or token is reported in the form rather
// than leaving the app disconnected.
export interface ExternalNodeApi {
  get(): Promise<ExternalNodeConfig | null>
  set(cfg: ExternalNodeConfig): Promise<Result<void>>
  clear(): Promise<Result<void>>
}

// --- Live feed (#44) ---

// FeedApi streams block events from the node. subscribe starts a server stream
// for an audience ("" = all the node follows); onEvent registers a listener and
// returns an unsubscribe function.
export interface FeedApi {
  subscribe(audienceCode: string): Promise<Result<void>>
  unsubscribe(): Promise<void>
  onEvent(cb: (summary: BlockSummary) => void): () => void
}

// --- Connections (#46) ---

export interface IdentityCard {
  address: string
  kyberPub: string
  mldsaPub: string
}

export interface ConnectionInfo {
  peer: string
  epoch: number
  audienceCode: string
}

export interface PrivateMessage {
  author: string
  text: string
  timestamp: number
}

export interface ConnectionsApi {
  getIdentity(): Promise<Result<IdentityCard>>
  addPeer(card: IdentityCard): Promise<Result<void>>
  start(address: string): Promise<Result<{ requestId: string }>>
  list(): Promise<Result<ConnectionInfo[]>>
  rotate(address: string): Promise<Result<void>>
  close(address: string): Promise<Result<void>>
  sendText(address: string, text: string): Promise<Result<{ id: string }>>
  messages(address: string): Promise<Result<PrivateMessage[]>>
}

// --- Identity & keys (#47) ---

// IdentityApi exposes the dangerous, confirmation-gated key operations. The
// renderer must pass confirm: true (after an explicit user confirmation); the
// node holds the keys and enforces the gate.
export interface IdentityApi {
  rotate(newPassphrase: string, confirm: boolean): Promise<Result<{ identity: string }>>
  burn(notice: string, confirm: boolean): Promise<Result<{ blockId: string }>>
}

export interface BpDesktop {
  versions: () => { electron: string; chrome: string; node: string }
  node: NodeApi
  lifecycle: LifecycleApi
  externalNode: ExternalNodeApi
  feed: FeedApi
  connections: ConnectionsApi
  identity: IdentityApi
}

// IPC channel names.
export const NODE_CHANNELS = {
  getStatus: 'node:getStatus',
  bootstrapWorld: 'node:bootstrapWorld',
  unlockKeystore: 'node:unlockKeystore',
  clearKeystore: 'node:clearKeystore',
  postText: 'node:postText',
  getBlock: 'node:getBlock',
  listBlocks: 'node:listBlocks',
} as const

export const LIFECYCLE_CHANNELS = {
  getState: 'lifecycle:getState',
  recentLogs: 'lifecycle:recentLogs',
} as const

export const EXTERNAL_NODE_CHANNELS = {
  get: 'external:get',
  set: 'external:set',
  clear: 'external:clear',
} as const

export const FEED_CHANNELS = {
  subscribe: 'feed:subscribe',
  unsubscribe: 'feed:unsubscribe',
  event: 'feed:event', // main → renderer push
} as const

export const CONNECTION_CHANNELS = {
  getIdentity: 'conn:getIdentity',
  addPeer: 'conn:addPeer',
  start: 'conn:start',
  list: 'conn:list',
  rotate: 'conn:rotate',
  close: 'conn:close',
  sendText: 'conn:sendText',
  messages: 'conn:messages',
} as const

export const IDENTITY_CHANNELS = {
  rotate: 'identity:rotate',
  burn: 'identity:burn',
} as const
