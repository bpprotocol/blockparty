import { mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import path from 'node:path'
import type { ExternalNodeConfig, Result } from './bridge'
import type { ExternalNode } from './node-config'
import type { NodeConnection } from './node-supervisor'

// Where a client-only build remembers the node the user pointed it at. Kept in
// the app's userData dir, next to nothing else the renderer can reach: the
// token stays in the main process, like the managed node's api.token (#29).
const FILE = 'external-node.json'

export function externalNodePath(userDataDir: string): string {
  return path.join(userDataDir, FILE)
}

// loadExternalNode reads the saved endpoint, or null when there is none (or the
// file is unreadable/corrupt — a bad file must not stop the app from starting).
export function loadExternalNode(userDataDir: string): ExternalNode | null {
  try {
    const raw = JSON.parse(readFileSync(externalNodePath(userDataDir), 'utf8')) as ExternalNode
    if (!raw || typeof raw.baseUrl !== 'string' || !raw.baseUrl) return null
    return {
      baseUrl: raw.baseUrl,
      token: typeof raw.token === 'string' ? raw.token : undefined,
      dataDir: typeof raw.dataDir === 'string' ? raw.dataDir : undefined,
    }
  } catch {
    return null
  }
}

export function saveExternalNode(userDataDir: string, cfg: ExternalNode): void {
  mkdirSync(userDataDir, { recursive: true })
  writeFileSync(externalNodePath(userDataDir), JSON.stringify(cfg, null, 2), { mode: 0o600 })
}

export function clearExternalNode(userDataDir: string): void {
  rmSync(externalNodePath(userDataDir), { force: true })
}

// normalizeBaseUrl accepts what a user would type — "127.0.0.1:4400" — as well
// as a full URL.
export function normalizeBaseUrl(addr: string): string {
  const a = addr.trim()
  if (!a) return ''
  return a.startsWith('http://') || a.startsWith('https://') ? a : `http://${a}`
}

// ConnectDeps are the moving parts of connecting to an external node, injected
// so the flow can be tested without Electron or a real node.
export interface ConnectDeps {
  // attach points the supervisor at the endpoint and returns how to reach it.
  attach(cfg: ExternalNode): NodeConnection
  // probe checks the node answers before anything is saved or swapped in.
  probe(conn: NodeConnection): Promise<Result<unknown>>
  save(cfg: ExternalNode): void
  // use swaps the app's client (and live feed) over to the connection.
  use(conn: NodeConnection): void
}

// connectExternalNode validates an endpoint, then persists it and switches the
// app over. A node that does not answer is reported back to the setup form and
// nothing is saved — a bad address must not leave the app pointing at it (#51).
export async function connectExternalNode(
  cfg: ExternalNodeConfig,
  deps: ConnectDeps,
): Promise<Result<void>> {
  const baseUrl = normalizeBaseUrl(cfg.baseUrl)
  if (!baseUrl) return { ok: false, error: 'Enter the node API address.' }

  const target: ExternalNode = { baseUrl, token: cfg.token, dataDir: cfg.dataDir }
  const conn = deps.attach(target)
  const probed = await deps.probe(conn)
  if (!probed.ok) return { ok: false, error: probed.error }

  deps.save(target)
  deps.use(conn)
  return { ok: true, value: undefined }
}
