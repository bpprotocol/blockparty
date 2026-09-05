import { app, BrowserWindow, Menu, net, protocol, shell } from 'electron'
import path from 'node:path'
import { pathToFileURL } from 'node:url'
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
  registerWindowIpc,
  forwardMaximizeState,
} from './ipc'
import { resolveSupervisorOptions } from './node-config'
import { NodeClient, nodeTransport } from './node-client'
import { NodeSupervisor } from './node-supervisor'

const isDev = process.env.NODE_ENV === 'development'

// The generated SPA is served over a custom scheme rather than file://, for two
// reasons: Nuxt emits absolute asset paths (/_nuxt/...), which file:// resolves
// against the filesystem root, and its entry is an ES module, which browsers
// refuse to load cross-origin over file://. Both work over a standard scheme.
const APP_SCHEME = 'app'
const APP_ORIGIN = `${APP_SCHEME}://bundle`
const RENDERER_ROOT = path.join(__dirname, '../.output/public')

// Must run before the app is ready: marks the scheme standard and secure, so
// module scripts, fetch and the usual web platform rules apply.
protocol.registerSchemesAsPrivileged([
  {
    scheme: APP_SCHEME,
    privileges: { standard: true, secure: true, supportFetchAPI: true, corsEnabled: true },
  },
])

// registerAppProtocol serves the static bundle. Paths are resolved inside
// RENDERER_ROOT and anything outside it is refused, so a crafted URL cannot
// read the rest of the disk. Unknown paths fall back to index.html, the SPA
// entry point.
function registerAppProtocol(): void {
  protocol.handle(APP_SCHEME, async (request) => {
    const { pathname } = new URL(request.url)
    const decoded = decodeURIComponent(pathname)
    const target = path.join(RENDERER_ROOT, decoded)
    const withinRoot = path.resolve(target).startsWith(path.resolve(RENDERER_ROOT) + path.sep)
    const file =
      withinRoot && path.extname(decoded) ? target : path.join(RENDERER_ROOT, 'index.html')
    return net.fetch(pathToFileURL(file).toString())
  })
}

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

// installAppMenu drops Electron's default File/Edit/View/Window/Help bar: the
// app's own left rail is its navigation, and none of those items did anything
// this app needs.
//
// macOS is the exception — the system always shows a menu bar for the focused
// app, and the standard shortcuts (Cmd+Q, Cmd+C/V, Cmd+W) come from it, so
// there it keeps a minimal one built from the standard roles.
function installAppMenu(): void {
  if (process.platform === 'darwin') {
    Menu.setApplicationMenu(
      Menu.buildFromTemplate([{ role: 'appMenu' }, { role: 'editMenu' }, { role: 'windowMenu' }]),
    )
    return
  }
  Menu.setApplicationMenu(null)
}

// Toggle DevTools without the View menu that used to carry it. Development
// only: a packaged build should not open DevTools on a stray keypress.
function enableDevToolsShortcut(win: BrowserWindow): void {
  win.webContents.on('before-input-event', (_event, input) => {
    const toggle =
      input.key === 'F12' ||
      (input.control && input.shift && input.key.toLowerCase() === 'i') ||
      (input.meta && input.alt && input.key.toLowerCase() === 'i')
    if (input.type === 'keyDown' && toggle) win.webContents.toggleDevTools()
  })
}

function createMainWindow(): BrowserWindow {
  const win = new BrowserWindow({
    width: 1100,
    height: 760,
    show: false,
    // Frameless: the renderer draws its own title bar (the navbar doubles as
    // the drag region). macOS keeps its traffic lights via hiddenInset, so
    // only Windows/Linux need the app's own window buttons.
    frame: process.platform === 'darwin',
    titleBarStyle: process.platform === 'darwin' ? 'hiddenInset' : 'default',
    // Below the frameless chrome the page paints its own background; this
    // avoids a white flash before the renderer's first paint.
    backgroundColor: '#f2f3f7',
    // Belt and braces on Windows/Linux: with no menu set there is nothing to
    // show, and this keeps the Alt key from summoning one.
    autoHideMenuBar: true,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true,
    },
  })
  win.once('ready-to-show', () => win.show())
  forwardMaximizeState(win)
  if (isDev) enableDevToolsShortcut(win)
  win.webContents.setWindowOpenHandler(({ url }) => {
    void shell.openExternal(url)
    return { action: 'deny' }
  })
  if (isDev) {
    void win.loadURL(process.env.ELECTRON_RENDERER_URL ?? 'http://localhost:3000')
  } else {
    void win.loadURL(`${APP_ORIGIN}/index.html`)
  }
  return win
}

void app.whenReady().then(async () => {
  installAppMenu()
  registerWindowIpc()
  if (!isDev) registerAppProtocol()

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
