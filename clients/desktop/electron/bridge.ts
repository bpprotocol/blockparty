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
  endpoint: string
  restarts: number
}

export interface LifecycleApi {
  getState(): Promise<LifecycleState>
  recentLogs(): Promise<string[]>
}

export interface BpDesktop {
  versions: () => { electron: string; chrome: string; node: string }
  node: NodeApi
  lifecycle: LifecycleApi
}

// IPC channel names.
export const NODE_CHANNELS = {
  getStatus: 'node:getStatus',
  bootstrapWorld: 'node:bootstrapWorld',
  postText: 'node:postText',
  getBlock: 'node:getBlock',
  listBlocks: 'node:listBlocks',
} as const

export const LIFECYCLE_CHANNELS = {
  getState: 'lifecycle:getState',
  recentLogs: 'lifecycle:recentLogs',
} as const
