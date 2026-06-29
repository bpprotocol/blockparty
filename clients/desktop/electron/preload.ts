import { contextBridge, ipcRenderer, type IpcRendererEvent } from 'electron'
import {
  FEED_CHANNELS,
  LIFECYCLE_CHANNELS,
  NODE_CHANNELS,
  type BlockSummary,
  type BpDesktop,
  type FeedApi,
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

const feed: FeedApi = {
  subscribe: (audienceCode) => ipcRenderer.invoke(FEED_CHANNELS.subscribe, audienceCode),
  unsubscribe: () => ipcRenderer.invoke(FEED_CHANNELS.unsubscribe),
  onEvent: (cb) => {
    const handler = (_e: IpcRendererEvent, summary: BlockSummary): void => cb(summary)
    ipcRenderer.on(FEED_CHANNELS.event, handler)
    return () => ipcRenderer.removeListener(FEED_CHANNELS.event, handler)
  },
}

const api: BpDesktop = {
  versions: () => ({
    electron: process.versions.electron ?? '',
    chrome: process.versions.chrome ?? '',
    node: process.versions.node ?? '',
  }),
  node,
  lifecycle,
  feed,
}

contextBridge.exposeInMainWorld('bpDesktop', api)
