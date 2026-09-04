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
  keystoreExists: false,
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
  it('shows unlock when a keystore exists but no World is loaded', () => {
    expect(
      deriveView({
        hasBridge: true,
        connected: true,
        status: { ...baseStatus, keystoreExists: true },
      }),
    ).toBe('unlock')
  })
  it('prefers ready over unlock once the keystore is open', () => {
    expect(
      deriveView({
        hasBridge: true,
        connected: true,
        status: { ...baseStatus, keystoreExists: true, worldLoaded: true },
      }),
    ).toBe('ready')
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
function installBridge(
  over: Partial<{
    status: NodeStatus
    bootstrapOk: boolean
    unlockOk: boolean
    clearOk: boolean
  }> = {},
) {
  const status = over.status ?? baseStatus
  const bootstrapWorld = vi.fn(async () =>
    over.bootstrapOk === false
      ? { ok: false as const, error: 'bad seed' }
      : { ok: true as const, value: { world: 'w1', identity: 'i1' } },
  )
  const clearKeystore = vi.fn(async () =>
    over.clearOk === false
      ? { ok: false as const, error: 'core: clear keystore: keystore: does not exist' }
      : { ok: true as const, value: undefined },
  )
  const unlockKeystore = vi.fn(async () =>
    over.unlockOk === false
      ? { ok: false as const, error: 'keystore: incorrect passphrase or corrupt keystore' }
      : { ok: true as const, value: { world: 'w1', identity: 'i1' } },
  )
  const getStatus = vi.fn(async () => ({ ok: true as const, value: status }))
  ;(globalThis as { window?: unknown }).window = {
    bpDesktop: {
      node: {
        getStatus,
        bootstrapWorld,
        unlockKeystore,
        clearKeystore,
        postText: vi.fn(),
        getBlock: vi.fn(),
        listBlocks: vi.fn(),
      },
      lifecycle: { getState: vi.fn(async () => ({})), recentLogs: vi.fn(async () => []) },
      versions: () => ({ electron: '', chrome: '', node: '' }),
    },
  }
  return { bootstrapWorld, unlockKeystore, clearKeystore, getStatus }
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

  it('routes to unlock when the node already holds a keystore', async () => {
    installBridge({ status: { ...baseStatus, keystoreExists: true } })
    const n = useNode()
    await n.refresh()
    expect(n.view.value).toBe('unlock')
  })

  it('unlock forwards the passphrase and advances to ready on success', async () => {
    const { unlockKeystore, getStatus } = installBridge({
      status: { ...baseStatus, keystoreExists: true },
    })
    // After unlocking, the node reports a loaded World.
    getStatus.mockResolvedValue({
      ok: true,
      value: { ...baseStatus, keystoreExists: true, worldLoaded: true, canAuthor: true },
    })
    const n = useNode()
    const res = await n.unlock('pw')
    expect(res.ok).toBe(true)
    expect(unlockKeystore).toHaveBeenCalledWith('pw')
    expect(n.view.value).toBe('ready')
  })

  it('unlock surfaces a wrong passphrase and stays on the unlock view', async () => {
    installBridge({ status: { ...baseStatus, keystoreExists: true }, unlockOk: false })
    const n = useNode()
    await n.refresh()
    const res = await n.unlock('nope')
    expect(res.ok).toBe(false)
    expect(res.error).toContain('incorrect passphrase')
    expect(n.view.value).toBe('unlock')
  })

  it('unlock rejects an empty passphrase without calling the node', async () => {
    const { unlockKeystore } = installBridge({ status: { ...baseStatus, keystoreExists: true } })
    const n = useNode()
    const res = await n.unlock('')
    expect(res.ok).toBe(false)
    expect(unlockKeystore).not.toHaveBeenCalled()
  })

  it('clearKeystore confirms explicitly and falls through to onboarding', async () => {
    const { clearKeystore, getStatus } = installBridge({
      status: { ...baseStatus, keystoreExists: true },
    })
    const n = useNode()
    await n.refresh()
    expect(n.view.value).toBe('unlock')

    // After clearing, the node reports no keystore.
    getStatus.mockResolvedValue({ ok: true, value: baseStatus })
    const res = await n.clearKeystore()
    expect(res.ok).toBe(true)
    // The node's confirmation gate (#29) is only satisfied with confirm: true.
    expect(clearKeystore).toHaveBeenCalledWith(true)
    expect(n.view.value).toBe('onboarding')
  })

  it('clearKeystore surfaces the node error and stays on unlock', async () => {
    installBridge({ status: { ...baseStatus, keystoreExists: true }, clearOk: false })
    const n = useNode()
    await n.refresh()
    const res = await n.clearKeystore()
    expect(res.ok).toBe(false)
    expect(res.error).toContain('does not exist')
    expect(n.view.value).toBe('unlock')
  })

  it('bootstrap surfaces the node error on failure', async () => {
    installBridge({ bootstrapOk: false })
    const n = useNode()
    const res = await n.bootstrap({ worldSeed: '', identityPassphrase: '', keystorePassphrase: '' })
    expect(res.ok).toBe(false)
    expect(res.error).toBe('bad seed')
  })
})
