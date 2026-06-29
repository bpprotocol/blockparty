import { ipcMain } from 'electron'
import {
  NODE_CHANNELS,
  type BootstrapRequest,
  type ListFilter,
  type NodeApi,
  type PostTextRequest,
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
