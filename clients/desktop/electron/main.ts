import { app, BrowserWindow, shell } from 'electron'
import path from 'node:path'
import type { NodeApi, Result } from './bridge'
import { registerLifecycleIpc, registerNodeIpc } from './ipc'
import { resolveSupervisorOptions } from './node-config'
import { NodeClient, nodeTransport } from './node-client'
import { NodeSupervisor } from './node-supervisor'

const isDev = process.env.NODE_ENV === 'development'

let supervisor: NodeSupervisor | undefined
let quitting = false

// disconnectedNodeApi answers every call with a connection error, used when the
// node could not be started/attached so the UI shows a disconnected state.
function disconnectedNodeApi(reason: string): NodeApi {
  const fail = async (): Promise<Result<never>> => ({ ok: false, error: reason })
  return {
    getStatus: fail,
    bootstrapWorld: fail,
    postText: fail,
    getBlock: fail,
    listBlocks: fail,
  }
}

function createWindow(): void {
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
}

async function startNode(): Promise<NodeApi> {
  supervisor = new NodeSupervisor(
    resolveSupervisorOptions(app.getPath('appData'), process.resourcesPath),
  )
  registerLifecycleIpc(supervisor)
  try {
    const conn = await supervisor.start()
    return new NodeClient(nodeTransport(conn.baseUrl, conn.token))
  } catch (e) {
    return disconnectedNodeApi(`node unavailable: ${e instanceof Error ? e.message : String(e)}`)
  }
}

void app.whenReady().then(async () => {
  // Manage (or attach to) the local node, then expose it to the renderer (#42).
  registerNodeIpc(await startNode())

  createWindow()
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow()
  })
})

// Stop a managed node cleanly before the app exits.
app.on('before-quit', (e) => {
  if (quitting || !supervisor) return
  e.preventDefault()
  quitting = true
  void supervisor.stop().finally(() => app.quit())
})

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit()
})
