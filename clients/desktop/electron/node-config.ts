import { accessSync, constants, statSync } from 'node:fs'
import path from 'node:path'
import type { LifecycleMode } from './bridge'
import type { SupervisorOptions } from './node-supervisor'

function toBaseUrl(addr: string): string {
  return addr.startsWith('http://') || addr.startsWith('https://') ? addr : `http://${addr}`
}

function bpnodeBinName(): string {
  return process.platform === 'win32' ? 'bpnode.exe' : 'bpnode'
}

// hasBundledNode reports whether a runnable bpnode for *this* platform sits at
// binPath. A client-only package ships without one, which is what makes managed
// mode unavailable (#51) — the presence of the binary decides, not an env var.
export function hasBundledNode(binPath: string): boolean {
  try {
    if (!statSync(binPath).isFile()) return false
  } catch {
    return false
  }
  // The executable bit is meaningless on Windows, where extension decides.
  if (process.platform === 'win32') return true
  try {
    accessSync(binPath, constants.X_OK)
    return true
  } catch {
    return false
  }
}

// ExternalNode is a user-configured endpoint for a node this app does not run:
// the base URL plus either a pasted bearer token or the node's data dir to read
// api.token from.
export interface ExternalNode {
  baseUrl: string
  token?: string
  dataDir?: string
}

// resolveSupervisorOptions decides how the app runs the node.
//
// Managed mode spawns the bundled bpnode and stays the default *when that
// binary is present*. A client-only build (#51) ships without it, so there is
// nothing to spawn: the app resolves to attach-only and connects to a node the
// user runs elsewhere. BPNODE_ATTACH=1 forces attach either way, and
// BPNODE_BIN/BPNODE_API_ADDR/BPNODE_DATA_DIR still override for development.
export function resolveSupervisorOptions(
  appDataDir: string,
  resourcesPath: string,
  external?: ExternalNode | null,
): SupervisorOptions {
  const binPath = process.env.BPNODE_BIN ?? path.join(resourcesPath, bpnodeBinName())
  const canManage = hasBundledNode(binPath)

  // Attach when asked to, and always when there is no binary to manage.
  const mode: LifecycleMode = process.env.BPNODE_ATTACH === '1' || !canManage ? 'attach' : 'managed'

  // Endpoint precedence: env (development) → saved external node → the local
  // default. In attach-only mode the default is only a placeholder: without an
  // env or saved endpoint the app asks the user for one instead of attaching.
  const envAddr = process.env.BPNODE_API_ADDR
  const savedUrl = mode === 'attach' ? external?.baseUrl : undefined
  const apiAddr = envAddr ?? savedUrl ?? '127.0.0.1:4400'
  // BPNODE_ATTACH=1 is itself a decision to attach — to the local default when
  // no address is given — so development keeps working without a saved node.
  const attachConfigured =
    mode === 'managed' || process.env.BPNODE_ATTACH === '1' || !!envAddr || !!savedUrl

  const envDataDir = process.env.BPNODE_DATA_DIR
  const dataDir =
    envDataDir ??
    (mode === 'attach' ? external?.dataDir : undefined) ??
    path.join(appDataDir, 'blockparty', 'node')

  return {
    mode,
    canManage,
    attachConfigured,
    baseUrl: toBaseUrl(apiAddr),
    apiAddr,
    dataDir,
    binPath,
    nodeMode: process.env.BPNODE_MODE ?? 'personal',
    token: mode === 'attach' ? (external?.token ?? '') : '',
  }
}
