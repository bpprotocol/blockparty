import { contextBridge, ipcRenderer, type IpcRendererEvent } from 'electron'
import {
  CONNECTION_CHANNELS,
  FEED_CHANNELS,
  LIFECYCLE_CHANNELS,
  NODE_CHANNELS,
  type BlockSummary,
  type BpDesktop,
  type ConnectionsApi,
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

const connections: ConnectionsApi = {
  getIdentity: () => ipcRenderer.invoke(CONNECTION_CHANNELS.getIdentity),
  addPeer: (card) => ipcRenderer.invoke(CONNECTION_CHANNELS.addPeer, card),
  start: (address) => ipcRenderer.invoke(CONNECTION_CHANNELS.start, address),
  list: () => ipcRenderer.invoke(CONNECTION_CHANNELS.list),
  rotate: (address) => ipcRenderer.invoke(CONNECTION_CHANNELS.rotate, address),
  close: (address) => ipcRenderer.invoke(CONNECTION_CHANNELS.close, address),
  sendText: (address, text) => ipcRenderer.invoke(CONNECTION_CHANNELS.sendText, address, text),
  messages: (address) => ipcRenderer.invoke(CONNECTION_CHANNELS.messages, address),
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
  connections,
}

contextBridge.exposeInMainWorld('bpDesktop', api)
