import { ipcMain } from 'electron'
import {
  FEED_CHANNELS,
  LIFECYCLE_CHANNELS,
  NODE_CHANNELS,
  type BootstrapRequest,
  type LifecycleState,
  type ListFilter,
  type NodeApi,
  type PostTextRequest,
  type Result,
} from './bridge'

// registerNodeIpc exposes the node API to the renderer over IPC. Each handler
// delegates to the main-process NodeClient (which holds the bearer token).
export function registerNodeIpc(api: NodeApi): void {
  ipcMain.handle(NODE_CHANNELS.getStatus, () => api.getStatus())
  ipcMain.handle(NODE_CHANNELS.bootstrapWorld, (_e, req: BootstrapRequest) =>
    api.bootstrapWorld(req),
  )
  ipcMain.handle(NODE_CHANNELS.postText, (_e, req: PostTextRequest) => api.postText(req))
  ipcMain.handle(NODE_CHANNELS.getBlock, (_e, id: string) => api.getBlock(id))
  ipcMain.handle(NODE_CHANNELS.listBlocks, (_e, filter: ListFilter) => api.listBlocks(filter))
}

// Lifecycle is the subset of the supervisor exposed to the renderer.
export interface Lifecycle {
  getState(): LifecycleState
  recentLogs(): string[]
}

// registerLifecycleIpc exposes node process state + recent logs (#42).
export function registerLifecycleIpc(supervisor: Lifecycle): void {
  ipcMain.handle(LIFECYCLE_CHANNELS.getState, () => supervisor.getState())
  ipcMain.handle(LIFECYCLE_CHANNELS.recentLogs, () => supervisor.recentLogs())
}

// Feed is the subset of the feed manager exposed to the renderer (#44).
export interface Feed {
  subscribe(audienceCode: string): Result<void>
  unsubscribe(): void
}

// registerFeedIpc exposes the live block feed; a nil feed (no node) answers with
// a connection error.
export function registerFeedIpc(getFeed: () => Feed | undefined): void {
  ipcMain.handle(FEED_CHANNELS.subscribe, (_e, audienceCode: string): Result<void> => {
    const feed = getFeed()
    if (!feed) return { ok: false, error: 'node not connected' }
    return feed.subscribe(audienceCode)
  })
  ipcMain.handle(FEED_CHANNELS.unsubscribe, () => {
    getFeed()?.unsubscribe()
  })
}
