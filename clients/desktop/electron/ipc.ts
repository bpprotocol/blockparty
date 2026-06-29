import { ipcMain } from 'electron'
import {
  LIFECYCLE_CHANNELS,
  NODE_CHANNELS,
  type BootstrapRequest,
  type LifecycleState,
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
