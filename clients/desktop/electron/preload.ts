import { contextBridge, ipcRenderer } from 'electron'
import {
  LIFECYCLE_CHANNELS,
  NODE_CHANNELS,
  type BpDesktop,
  type LifecycleApi,
  type NodeApi,
} from './bridge'

// The node API surface, forwarded to the main process over IPC. The renderer
// never holds the bearer token or talks to the node directly.
const node: NodeApi = {
  getStatus: () => ipcRenderer.invoke(NODE_CHANNELS.getStatus),
  bootstrapWorld: (req) => ipcRenderer.invoke(NODE_CHANNELS.bootstrapWorld, req),
  postText: (req) => ipcRenderer.invoke(NODE_CHANNELS.postText, req),
  getBlock: (id) => ipcRenderer.invoke(NODE_CHANNELS.getBlock, id),
  listBlocks: (filter) => ipcRenderer.invoke(NODE_CHANNELS.listBlocks, filter),
}

const lifecycle: LifecycleApi = {
  getState: () => ipcRenderer.invoke(LIFECYCLE_CHANNELS.getState),
  recentLogs: () => ipcRenderer.invoke(LIFECYCLE_CHANNELS.recentLogs),
}

const api: BpDesktop = {
  versions: () => ({
    electron: process.versions.electron ?? '',
    chrome: process.versions.chrome ?? '',
    node: process.versions.node ?? '',
  }),
  node,
  lifecycle,
}

contextBridge.exposeInMainWorld('bpDesktop', api)
