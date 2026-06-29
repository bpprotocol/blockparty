import path from 'node:path'
import type { LifecycleMode } from './bridge'
import type { SupervisorOptions } from './node-supervisor'

function toBaseUrl(addr: string): string {
  return addr.startsWith('http://') || addr.startsWith('https://') ? addr : `http://${addr}`
}

function bpnodeBinName(): string {
  return process.platform === 'win32' ? 'bpnode.exe' : 'bpnode'
}

// resolveSupervisorOptions decides how the app runs the node. Managed mode
// (default) spawns the bundled bpnode; attach mode (BPNODE_ATTACH=1) connects to
// an externally-run daemon. All knobs are env-overridable for development.
export function resolveSupervisorOptions(
  appDataDir: string,
  resourcesPath: string,
): SupervisorOptions {
  const mode: LifecycleMode = process.env.BPNODE_ATTACH === '1' ? 'attach' : 'managed'
  const apiAddr = process.env.BPNODE_API_ADDR ?? '127.0.0.1:4400'
  const dataDir = process.env.BPNODE_DATA_DIR ?? path.join(appDataDir, 'blockparty', 'node')
  const binPath = process.env.BPNODE_BIN ?? path.join(resourcesPath, bpnodeBinName())
  const nodeMode = process.env.BPNODE_MODE ?? 'personal'
  return {
    mode,
    baseUrl: toBaseUrl(apiAddr),
    apiAddr,
    dataDir,
    binPath,
    nodeMode,
  }
}
