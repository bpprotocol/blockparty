import { describe, expect, it, vi } from 'vitest'
import { mkdtempSync, rmSync, statSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import path from 'node:path'
import {
  clearExternalNode,
  connectExternalNode,
  externalNodePath,
  loadExternalNode,
  normalizeBaseUrl,
  saveExternalNode,
} from './external-node'
import type { ConnectDeps } from './external-node'
import type { NodeConnection } from './node-supervisor'

function tmpDir(): string {
  return mkdtempSync(path.join(tmpdir(), 'bpext-'))
}

describe('external node config', () => {
  it('round-trips the saved endpoint', () => {
    const dir = tmpDir()
    saveExternalNode(dir, { baseUrl: 'http://10.0.0.5:4400', token: 'tok' })
    expect(loadExternalNode(dir)).toEqual({
      baseUrl: 'http://10.0.0.5:4400',
      token: 'tok',
      dataDir: undefined,
    })
    rmSync(dir, { recursive: true, force: true })
  })

  it('returns null when nothing is saved', () => {
    const dir = tmpDir()
    expect(loadExternalNode(dir)).toBeNull()
    rmSync(dir, { recursive: true, force: true })
  })

  it('returns null for a corrupt or incomplete file rather than throwing', () => {
    const dir = tmpDir()
    writeFileSync(externalNodePath(dir), 'not json')
    expect(loadExternalNode(dir)).toBeNull()
    writeFileSync(externalNodePath(dir), JSON.stringify({ token: 'orphan' }))
    expect(loadExternalNode(dir)).toBeNull()
    rmSync(dir, { recursive: true, force: true })
  })

  it('keeps the token out of world-readable files', () => {
    if (process.platform === 'win32') return
    const dir = tmpDir()
    saveExternalNode(dir, { baseUrl: 'http://10.0.0.5:4400', token: 'tok' })
    expect(statSync(externalNodePath(dir)).mode & 0o077).toBe(0)
    rmSync(dir, { recursive: true, force: true })
  })

  it('clear removes the file and is safe when there is none', () => {
    const dir = tmpDir()
    saveExternalNode(dir, { baseUrl: 'http://10.0.0.5:4400' })
    clearExternalNode(dir)
    expect(loadExternalNode(dir)).toBeNull()
    clearExternalNode(dir) // no throw
    rmSync(dir, { recursive: true, force: true })
  })
})

describe('normalizeBaseUrl', () => {
  it('accepts a host:port and a full URL', () => {
    expect(normalizeBaseUrl('127.0.0.1:4400')).toBe('http://127.0.0.1:4400')
    expect(normalizeBaseUrl(' https://node.example:443 ')).toBe('https://node.example:443')
    expect(normalizeBaseUrl('   ')).toBe('')
  })
})

describe('connectExternalNode', () => {
  // deps records what the flow did, standing in for the supervisor, a probe
  // against a real node, the on-disk config, and the live client swap.
  function deps(probeOk: boolean, error = 'connection refused') {
    const attach = vi.fn((cfg: { baseUrl: string; token?: string }): NodeConnection => ({
      baseUrl: cfg.baseUrl,
      token: cfg.token ?? 'from-data-dir',
    }))
    const probe = vi.fn(async () =>
      probeOk ? { ok: true as const, value: {} } : { ok: false as const, error },
    )
    const save = vi.fn()
    const use = vi.fn()
    return { attach, probe, save, use } satisfies ConnectDeps & Record<string, unknown>
  }

  it('normalizes, probes, saves and switches over on success', async () => {
    const d = deps(true)
    const r = await connectExternalNode({ baseUrl: '10.0.0.5:4400', token: 'tok' }, d)
    expect(r.ok).toBe(true)
    expect(d.attach).toHaveBeenCalledWith({
      baseUrl: 'http://10.0.0.5:4400',
      token: 'tok',
      dataDir: undefined,
    })
    expect(d.save).toHaveBeenCalledWith({
      baseUrl: 'http://10.0.0.5:4400',
      token: 'tok',
      dataDir: undefined,
    })
    expect(d.use).toHaveBeenCalledWith({ baseUrl: 'http://10.0.0.5:4400', token: 'tok' })
  })

  it('saves nothing when the node does not answer', async () => {
    const d = deps(false)
    const r = await connectExternalNode({ baseUrl: '10.0.0.9:4400', token: 'tok' }, d)
    expect(r.ok).toBe(false)
    if (!r.ok) expect(r.error).toContain('connection refused')
    expect(d.save).not.toHaveBeenCalled()
    expect(d.use).not.toHaveBeenCalled()
  })

  it('rejects an empty address without touching anything', async () => {
    const d = deps(true)
    const r = await connectExternalNode({ baseUrl: '  ' }, d)
    expect(r.ok).toBe(false)
    expect(d.attach).not.toHaveBeenCalled()
    expect(d.probe).not.toHaveBeenCalled()
  })

  it('passes a data dir through when no token is pasted', async () => {
    const d = deps(true)
    await connectExternalNode({ baseUrl: 'node.local:4400', dataDir: '/srv/bp' }, d)
    expect(d.attach).toHaveBeenCalledWith({
      baseUrl: 'http://node.local:4400',
      token: undefined,
      dataDir: '/srv/bp',
    })
    // The supervisor read api.token from that dir; the app uses what it got.
    expect(d.use).toHaveBeenCalledWith({
      baseUrl: 'http://node.local:4400',
      token: 'from-data-dir',
    })
  })
})
