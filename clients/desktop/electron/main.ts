import { app, BrowserWindow, shell } from 'electron'
import path from 'node:path'
import { FEED_CHANNELS, type ExternalNodeConfig, type NodeApi, type Result } from './bridge'
import {
  clearExternalNode,
  connectExternalNode,
  loadExternalNode,
  saveExternalNode,
} from './external-node'
import { FeedManager } from './feed-manager'
import {
  registerConnectionIpc,
  registerExternalNodeIpc,
  registerFeedIpc,
  registerIdentityIpc,
  registerLifecycleIpc,
  registerNodeIpc,
} from './ipc'
import { resolveSupervisorOptions } from './node-config'
import { NodeClient, nodeTransport } from './node-client'
import { NodeSupervisor } from './node-supervisor'

const isDev = process.env.NODE_ENV === 'development'

let supervisor: NodeSupervisor | undefined
let nodeClient: NodeClient | undefined
let nodeApi: NodeApi = disconnectedNodeApi('node not started')
let feed: FeedManager | undefined
let mainWindow: BrowserWindow | undefined
let quitting = false

// disconnectedNodeApi answers every call with a connection error, used when the
// node could not be started/attached so the UI shows a disconnected state.
function disconnectedNodeApi(reason: string): NodeApi {
  const fail = async (): Promise<Result<never>> => ({ ok: false, error: reason })
  return {
    getStatus: fail,
    bootstrapWorld: fail,
    unlockKeystore: fail,
    clearKeystore: fail,
    postText: fail,
    getBlock: fail,
    listBlocks: fail,
  }
}

async function startNode(): Promise<void> {
  // A client-only build (#51) has no bpnode to spawn, so the options resolve to
  // attach-only and carry any node the user configured earlier.
  supervisor = new NodeSupervisor(
    resolveSupervisorOptions(
      app.getPath('appData'),
      process.resourcesPath,
      loadExternalNode(app.getPath('userData')),
    ),
  )
  registerLifecycleIpc(supervisor)
  try {
    const conn = await supervisor.start()
    useConnection(conn.baseUrl, conn.token)
  } catch (e) {
    nodeClient = undefined
    nodeApi = disconnectedNodeApi(`node unavailable: ${e instanceof Error ? e.message : String(e)}`)
  }
}

// useConnection points the app's client (and the live feed) at a node. Called on
// startup and again whenever the user configures a different external node.
function useConnection(baseUrl: string, token: string): void {
  nodeClient = new NodeClient(nodeTransport(baseUrl, token))
  nodeApi = nodeClient
  feed?.unsubscribe()
  feed = new FeedManager(nodeClient, (summary) => {
    if (mainWindow && !mainWindow.isDestroyed()) {
      mainWindow.webContents.send(FEED_CHANNELS.event, summary)
    }
  })
}

// connectExternal validates an endpoint before saving it: a URL or token that
// doesn't answer is reported in the setup form, not left as a dead app.
async function connectExternal(cfg: ExternalNodeConfig): Promise<Result<void>> {
  const sup = supervisor
  if (!sup) return { ok: false, error: 'node supervisor unavailable' }
  return connectExternalNode(cfg, {
    attach: (target) => sup.attachTo(target),
    probe: (conn) => new NodeClient(nodeTransport(conn.baseUrl, conn.token)).getStatus(),
    save: (target) => saveExternalNode(app.getPath('userData'), target),
    use: (conn) => useConnection(conn.baseUrl, conn.token),
  })
}

function createMainWindow(): BrowserWindow {
  const win = new BrowserWindow({
    width: 1100,
    height: 760,
    show: false,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true,
    },
  })
  win.once('ready-to-show', () => win.show())
  win.webContents.setWindowOpenHandler(({ url }) => {
    void shell.openExternal(url)
    return { action: 'deny' }
  })
  if (isDev) {
    void win.loadURL(process.env.ELECTRON_RENDERER_URL ?? 'http://localhost:3000')
  } else {
    void win.loadFile(path.join(__dirname, '../.output/public/index.html'))
  }
  return win
}

void app.whenReady().then(async () => {
  // Manage (or attach to) the local node, then expose it to the renderer (#42).
  await startNode()
  registerNodeIpc(() => nodeApi)
  registerExternalNodeIpc({
    get: () => loadExternalNode(app.getPath('userData')),
    set: connectExternal,
    clear: async () => {
      clearExternalNode(app.getPath('userData'))
      return { ok: true, value: undefined }
    },
  })

  mainWindow = createMainWindow()

  registerFeedIpc(() => feed)
  registerConnectionIpc(() => nodeClient)
  registerIdentityIpc(() => nodeClient)

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) mainWindow = createMainWindow()
  })
})

// Stop a managed node cleanly before the app exits.
app.on('before-quit', (e) => {
  if (quitting || !supervisor) return
  e.preventDefault()
  quitting = true
  feed?.unsubscribe()
  void supervisor.stop().finally(() => app.quit())
})

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit()
})
