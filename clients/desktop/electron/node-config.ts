import { readFileSync } from 'node:fs'
import path from 'node:path'

export interface NodeConnection {
  baseUrl: string
  token: string
}

// resolveNodeConnection figures out where the local node's API is and reads its
// bearer token from the node data dir's api.token file (#29). Overridable by
// env for development; #42 (node lifecycle) provides these when it manages the
// bundled bpnode.
export function resolveNodeConnection(appDataDir: string): NodeConnection {
  const addr = process.env.BPNODE_API_ADDR ?? '127.0.0.1:4400'
  const baseUrl =
    addr.startsWith('http://') || addr.startsWith('https://') ? addr : `http://${addr}`

  const dataDir = process.env.BPNODE_DATA_DIR ?? path.join(appDataDir, 'blockparty', 'node')
  let token = ''
  try {
    token = readFileSync(path.join(dataDir, 'api.token'), 'utf8').trim()
  } catch {
    // Node not running / not yet configured — getStatus will report disconnected.
  }
  return { baseUrl, token }
}
