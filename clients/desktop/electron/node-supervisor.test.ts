import { describe, expect, it } from 'vitest'
import { chmodSync, mkdtempSync, rmSync } from 'node:fs'
import net from 'node:net'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { NodeSupervisor, NodeUnavailableError } from './node-supervisor'

const fakeNode = fileURLToPath(new URL('./testdata/fake-node.mjs', import.meta.url))
chmodSync(fakeNode, 0o755)

function tmpDir(): string {
  return mkdtempSync(path.join(tmpdir(), 'bpdesk-'))
}

function freePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const s = net.createServer()
    s.once('error', reject)
    s.listen(0, '127.0.0.1', () => {
      const port = (s.address() as net.AddressInfo).port
      s.close(() => resolve(port))
    })
  })
}

describe('NodeSupervisor', () => {
  it('spawns a managed node, becomes ready, and stops cleanly', async () => {
    const dataDir = tmpDir()
    const port = await freePort()
    const sup = new NodeSupervisor({
      mode: 'managed',
      baseUrl: `http://127.0.0.1:${port}`,
      apiAddr: `127.0.0.1:${port}`,
      dataDir,
      binPath: fakeNode,
      nodeMode: 'personal',
      readyTimeoutMs: 10_000,
    })

    const conn = await sup.start()
    expect(conn.token).toBe('fake-token')
    expect(sup.getState().state).toBe('running')
    expect(sup.recentLogs().some((l) => l.includes('node ready'))).toBe(true)

    await sup.stop()
    expect(sup.getState().state).toBe('stopped')

    rmSync(dataDir, { recursive: true, force: true })
  }, 15_000)

  it('attach mode does not spawn a process', async () => {
    const sup = new NodeSupervisor({
      mode: 'attach',
      baseUrl: 'http://127.0.0.1:65500',
      apiAddr: '127.0.0.1:65500',
      dataDir: tmpDir(),
      binPath: '/nonexistent/bpnode', // must not be spawned in attach mode
      nodeMode: 'personal',
    })
    const conn = await sup.start()
    expect(conn.baseUrl).toBe('http://127.0.0.1:65500')
    expect(sup.getState().mode).toBe('attach')
    expect(sup.getState().state).toBe('running')
    await sup.stop()
  })

  // A client-only build (#51) has no binary to run: it must say so, not attempt
  // a spawn that fails with ENOENT.
  it('refuses to spawn when no node ships with the build', async () => {
    const sup = new NodeSupervisor({
      mode: 'managed',
      canManage: false,
      baseUrl: 'http://127.0.0.1:65501',
      apiAddr: '127.0.0.1:65501',
      dataDir: tmpDir(),
      binPath: '/nonexistent/bpnode',
      nodeMode: 'personal',
    })
    await expect(sup.start()).rejects.toThrow(NodeUnavailableError)
    await expect(sup.start()).rejects.toThrow(/cannot run a node itself/)
    expect(sup.getState().canManage).toBe(false)
    expect(sup.getState().state).toBe('stopped')
  })

  it('asks for an endpoint when attach-only and none is configured', async () => {
    const sup = new NodeSupervisor({
      mode: 'attach',
      canManage: false,
      attachConfigured: false,
      baseUrl: 'http://127.0.0.1:4400',
      apiAddr: '127.0.0.1:4400',
      dataDir: tmpDir(),
      binPath: '/nonexistent/bpnode',
      nodeMode: 'personal',
    })
    await expect(sup.start()).rejects.toThrow(NodeUnavailableError)

    const state = sup.getState()
    expect(state.needsEndpoint).toBe(true)
    // Nothing to show the user as an endpoint until they pick one.
    expect(state.endpoint).toBe('')
  })

  it('attachTo points the app at a node and supplies its token', () => {
    const sup = new NodeSupervisor({
      mode: 'attach',
      canManage: false,
      attachConfigured: false,
      baseUrl: 'http://127.0.0.1:4400',
      apiAddr: '127.0.0.1:4400',
      dataDir: tmpDir(),
      binPath: '/nonexistent/bpnode',
      nodeMode: 'personal',
    })
    const conn = sup.attachTo({ baseUrl: 'http://10.0.0.5:4400', token: 'pasted-token' })
    expect(conn).toEqual({ baseUrl: 'http://10.0.0.5:4400', token: 'pasted-token' })

    const state = sup.getState()
    expect(state.needsEndpoint).toBe(false)
    expect(state.endpoint).toBe('http://10.0.0.5:4400')
    expect(state.state).toBe('running')
  })
})
