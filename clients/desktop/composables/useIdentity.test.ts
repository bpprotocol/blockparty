import { afterEach, describe, expect, it, vi } from 'vitest'
import { useIdentity } from './useIdentity'

function installBridge(over: { rotateOk?: boolean; burnOk?: boolean } = {}) {
  const rotate = vi.fn(async () =>
    over.rotateOk === false
      ? { ok: false as const, error: 'confirmation required' }
      : { ok: true as const, value: { identity: 'new-addr' } },
  )
  const burn = vi.fn(async () =>
    over.burnOk === false
      ? { ok: false as const, error: 'confirmation required' }
      : { ok: true as const, value: { blockId: 'burn-block' } },
  )
  ;(globalThis as { window?: unknown }).window = { bpDesktop: { identity: { rotate, burn } } }
  return { rotate, burn }
}

afterEach(() => {
  delete (globalThis as { window?: unknown }).window
})

describe('useIdentity', () => {
  it('does not rotate without a new passphrase', async () => {
    const { rotate } = installBridge()
    const id = useIdentity()
    expect(await id.rotate('', true)).toBeNull()
    expect(rotate).not.toHaveBeenCalled()
    expect(id.error.value).toBeTruthy()
  })

  it('rotates with confirmation and returns the new address', async () => {
    const { rotate } = installBridge()
    const id = useIdentity()
    expect(await id.rotate('new-pass', true)).toBe('new-addr')
    expect(rotate).toHaveBeenCalledWith('new-pass', true)
  })

  it('surfaces the gate error when the node rejects an unconfirmed op', async () => {
    installBridge({ burnOk: false })
    const id = useIdentity()
    expect(await id.burn('compromised', false)).toBeNull()
    expect(id.error.value).toBe('confirmation required')
  })

  it('burns with confirmation and returns the block id', async () => {
    const { burn } = installBridge()
    const id = useIdentity()
    expect(await id.burn('compromised', true)).toBe('burn-block')
    expect(burn).toHaveBeenCalledWith('compromised', true)
  })
})
