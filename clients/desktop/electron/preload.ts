import { contextBridge } from 'electron'

// The single, typed surface the renderer is allowed to call. It is intentionally
// tiny for the scaffold; the node API client (#41) extends it with connection +
// RPC plumbing. Nothing here exposes Node, fs, or the main process directly.
const api = {
  versions: (): { electron: string; chrome: string; node: string } => ({
    electron: process.versions.electron ?? '',
    chrome: process.versions.chrome ?? '',
    node: process.versions.node ?? '',
  }),
}

export type BpDesktopApi = typeof api

contextBridge.exposeInMainWorld('bpDesktop', api)
