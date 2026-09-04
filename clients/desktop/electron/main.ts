import { app, BrowserWindow, shell } from 'electron'
import path from 'node:path'
import { FEED_CHANNELS, type NodeApi, type Result } from './bridge'
import { FeedManager } from './feed-manager'
import {
  registerConnectionIpc,
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
let feed: FeedManager | undefined
let quitting = false

// disconnectedNodeApi answers every call with a connection error, used when the
// node could not be started/attached so the UI shows a disconnected state.
function disconnectedNodeApi(reason: string): NodeApi {
  const fail = async (): Promise<Result<never>> => ({ ok: false, error: reason })
  return {
    getStatus: fail,
    bootstrapWorld: fail,
    unlockKeystore: fail,
    postText: fail,
    getBlock: fail,
    listBlocks: fail,
  }
}

async function startNode(): Promise<NodeApi> {
  supervisor = new NodeSupervisor(
    resolveSupervisorOptions(app.getPath('appData'), process.resourcesPath),
  )
  registerLifecycleIpc(supervisor)
  try {
    const conn = await supervisor.start()
    nodeClient = new NodeClient(nodeTransport(conn.baseUrl, conn.token))
    return nodeClient
  } catch (e) {
    return disconnectedNodeApi(`node unavailable: ${e instanceof Error ? e.message : String(e)}`)
  }
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
  registerNodeIpc(await startNode())

  const win = createMainWindow()

  // The live feed (#44) pushes block events from the node to this window.
  if (nodeClient) {
    feed = new FeedManager(nodeClient, (summary) => {
      if (!win.isDestroyed()) win.webContents.send(FEED_CHANNELS.event, summary)
    })
  }
  registerFeedIpc(() => feed)
  registerConnectionIpc(() => nodeClient)
  registerIdentityIpc(() => nodeClient)

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createMainWindow()
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
