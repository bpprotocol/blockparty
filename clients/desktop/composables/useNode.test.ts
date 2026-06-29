import { afterEach, describe, expect, it, vi } from 'vitest'
import type { NodeStatus } from '../electron/bridge'
import { deriveView, useNode } from './useNode'

const baseStatus: NodeStatus = {
  version: '0.1.0',
  mode: 'personal',
  worldLoaded: false,
  world: '',
  identity: '',
  blockCount: 0,
  canAuthor: false,
}

describe('deriveView', () => {
  it('shows no-bridge without the preload bridge', () => {
    expect(deriveView({ hasBridge: false, connected: true, status: baseStatus })).toBe('no-bridge')
  })
  it('shows disconnected when not connected', () => {
    expect(deriveView({ hasBridge: true, connected: false, status: null })).toBe('disconnected')
  })
  it('shows onboarding when connected but no World loaded', () => {
    expect(deriveView({ hasBridge: true, connected: true, status: baseStatus })).toBe('onboarding')
  })
  it('shows ready when a World is loaded', () => {
    expect(
      deriveView({
        hasBridge: true,
        connected: true,
        status: { ...baseStatus, worldLoaded: true },
      }),
    ).toBe('ready')
  })
})

// fakeBridge installs a window.bpDesktop stub for the composable.
function installBridge(over: Partial<{ status: NodeStatus; bootstrapOk: boolean }> = {}) {
  const status = over.status ?? baseStatus
  const bootstrapWorld = vi.fn(async () =>
    over.bootstrapOk === false
      ? { ok: false as const, error: 'bad seed' }
      : { ok: true as const, value: { world: 'w1', identity: 'i1' } },
  )
  const getStatus = vi.fn(async () => ({ ok: true as const, value: status }))
  ;(globalThis as { window?: unknown }).window = {
    bpDesktop: {
      node: {
        getStatus,
        bootstrapWorld,
        postText: vi.fn(),
        getBlock: vi.fn(),
        listBlocks: vi.fn(),
      },
      lifecycle: { getState: vi.fn(async () => ({})), recentLogs: vi.fn(async () => []) },
      versions: () => ({ electron: '', chrome: '', node: '' }),
    },
  }
  return { bootstrapWorld, getStatus }
}

afterEach(() => {
  delete (globalThis as { window?: unknown }).window
})

describe('useNode', () => {
  it('refresh transitions to onboarding for an unconfigured node', async () => {
    installBridge()
    const n = useNode()
    await n.refresh()
    expect(n.connected.value).toBe(true)
    expect(n.view.value).toBe('onboarding')
  })

  it('bootstrap forwards secrets and advances to ready on success', async () => {
    const { bootstrapWorld, getStatus } = installBridge()
    // After bootstrap, the node reports a loaded World.
    getStatus.mockResolvedValue({ ok: true, value: { ...baseStatus, worldLoaded: true } })
    const n = useNode()
    const res = await n.bootstrap({
      worldSeed: 'seed',
      identityPassphrase: 'idp',
      keystorePassphrase: 'pw',
    })
    expect(res.ok).toBe(true)
    expect(bootstrapWorld).toHaveBeenCalledWith({
      worldSeed: 'seed',
      identityPassphrase: 'idp',
      keystorePassphrase: 'pw',
    })
    expect(n.view.value).toBe('ready')
  })

  it('bootstrap surfaces the node error on failure', async () => {
    installBridge({ bootstrapOk: false })
    const n = useNode()
    const res = await n.bootstrap({ worldSeed: '', identityPassphrase: '', keystorePassphrase: '' })
    expect(res.ok).toBe(false)
    expect(res.error).toBe('bad seed')
  })
})
