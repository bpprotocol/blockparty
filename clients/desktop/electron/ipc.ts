import { BrowserWindow, ipcMain } from 'electron'
import {
  CONNECTION_CHANNELS,
  EXTERNAL_NODE_CHANNELS,
  FEED_CHANNELS,
  IDENTITY_CHANNELS,
  LIFECYCLE_CHANNELS,
  NODE_CHANNELS,
  WINDOW_CHANNELS,
  type BootstrapRequest,
  type ExternalNodeConfig,
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
// getApi is read per call, so pointing the app at a different node (#51)
// swaps the client without re-registering handlers.
export function registerNodeIpc(getApi: () => NodeApi): void {
  ipcMain.handle(NODE_CHANNELS.getStatus, () => getApi().getStatus())
  ipcMain.handle(NODE_CHANNELS.bootstrapWorld, (_e, req: BootstrapRequest) =>
    getApi().bootstrapWorld(req),
  )
  ipcMain.handle(NODE_CHANNELS.unlockKeystore, (_e, keystorePassphrase: string) =>
    getApi().unlockKeystore(keystorePassphrase),
  )
  ipcMain.handle(NODE_CHANNELS.clearKeystore, (_e, confirm: boolean) =>
    getApi().clearKeystore(confirm),
  )
  ipcMain.handle(NODE_CHANNELS.postText, (_e, req: PostTextRequest) => getApi().postText(req))
  ipcMain.handle(NODE_CHANNELS.getBlock, (_e, id: string) => getApi().getBlock(id))
  ipcMain.handle(NODE_CHANNELS.listBlocks, (_e, filter: ListFilter) => getApi().listBlocks(filter))
}

// ExternalNodeStore is the main-process side of the external-node endpoint: it
// reads and writes the saved config and reconnects the app to it (#51).
export interface ExternalNodeStore {
  get(): ExternalNodeConfig | null
  set(cfg: ExternalNodeConfig): Promise<Result<void>>
  clear(): Promise<Result<void>>
}

// registerExternalNodeIpc exposes that store to the renderer.
export function registerExternalNodeIpc(store: ExternalNodeStore): void {
  ipcMain.handle(EXTERNAL_NODE_CHANNELS.get, () => store.get())
  ipcMain.handle(EXTERNAL_NODE_CHANNELS.set, (_e, cfg: ExternalNodeConfig) => store.set(cfg))
  ipcMain.handle(EXTERNAL_NODE_CHANNELS.clear, () => store.clear())
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

// registerWindowIpc lets the renderer drive its own frameless window. The
// window is resolved from the sender, so the handlers stay correct if the app
// ever has more than one.
export function registerWindowIpc(): void {
  const win = (e: Electron.IpcMainInvokeEvent): BrowserWindow | null =>
    BrowserWindow.fromWebContents(e.sender)

  ipcMain.handle(WINDOW_CHANNELS.minimize, (e) => {
    win(e)?.minimize()
  })
  ipcMain.handle(WINDOW_CHANNELS.toggleMaximize, (e): boolean => {
    const w = win(e)
    if (!w) return false
    if (w.isMaximized()) w.unmaximize()
    else w.maximize()
    return w.isMaximized()
  })
  ipcMain.handle(WINDOW_CHANNELS.close, (e) => {
    win(e)?.close()
  })
  ipcMain.handle(WINDOW_CHANNELS.isMaximized, (e): boolean => win(e)?.isMaximized() ?? false)
}

// forwardMaximizeState pushes maximize/restore to the renderer, so its button
// matches the window even when the change came from the window manager.
export function forwardMaximizeState(window: BrowserWindow): void {
  const send = (maximized: boolean): void => {
    if (!window.isDestroyed()) window.webContents.send(WINDOW_CHANNELS.maximizeChanged, maximized)
  }
  window.on('maximize', () => send(true))
  window.on('unmaximize', () => send(false))
}
