import { ipcMain } from 'electron'
import {
  CONNECTION_CHANNELS,
  FEED_CHANNELS,
  IDENTITY_CHANNELS,
  LIFECYCLE_CHANNELS,
  NODE_CHANNELS,
  type BootstrapRequest,
  type IdentityCard,
  type LifecycleState,
  type ListFilter,
  type NodeApi,
  type PostTextRequest,
  type Result,
} from './bridge'
import type { NodeClient } from './node-client'

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

const noClient = (): Result<never> => ({ ok: false, error: 'node not connected' })

// registerConnectionIpc exposes the connection RPCs (#46). A nil client (no
// node) answers with a connection error.
export function registerConnectionIpc(getClient: () => NodeClient | undefined): void {
  ipcMain.handle(CONNECTION_CHANNELS.getIdentity, () => getClient()?.getIdentity() ?? noClient())
  ipcMain.handle(
    CONNECTION_CHANNELS.addPeer,
    (_e, card: IdentityCard) => getClient()?.addPeer(card) ?? noClient(),
  )
  ipcMain.handle(
    CONNECTION_CHANNELS.start,
    (_e, address: string) => getClient()?.startConnection(address) ?? noClient(),
  )
  ipcMain.handle(CONNECTION_CHANNELS.list, () => getClient()?.listConnections() ?? noClient())
  ipcMain.handle(
    CONNECTION_CHANNELS.rotate,
    (_e, address: string) => getClient()?.rotateConnection(address) ?? noClient(),
  )
  ipcMain.handle(
    CONNECTION_CHANNELS.close,
    (_e, address: string) => getClient()?.closeConnection(address) ?? noClient(),
  )
  ipcMain.handle(
    CONNECTION_CHANNELS.sendText,
    (_e, address: string, text: string) =>
      getClient()?.sendPrivateText(address, text) ?? noClient(),
  )
  ipcMain.handle(
    CONNECTION_CHANNELS.messages,
    (_e, address: string) => getClient()?.connectionMessages(address) ?? noClient(),
  )
}

// registerIdentityIpc exposes the gated identity operations (#47).
export function registerIdentityIpc(getClient: () => NodeClient | undefined): void {
  ipcMain.handle(
    IDENTITY_CHANNELS.rotate,
    (_e, newPassphrase: string, confirm: boolean) =>
      getClient()?.rotateIdentity(newPassphrase, confirm) ?? noClient(),
  )
  ipcMain.handle(
    IDENTITY_CHANNELS.burn,
    (_e, notice: string, confirm: boolean) =>
      getClient()?.burnIdentity(notice, confirm) ?? noClient(),
  )
}
