import { afterEach, describe, expect, it, vi } from 'vitest'
import { useCompose, validatePost } from './useCompose'

describe('validatePost', () => {
  it('rejects empty text', () => {
    expect(validatePost(1, '   ')).toMatch(/write something/i)
  })
  it('rejects out-of-range audiences', () => {
    expect(validatePost(0, 'hi')).toMatch(/public audience/i)
    expect(validatePost(17, 'hi')).toMatch(/public audience/i)
  })
  it('accepts a valid intent', () => {
    expect(validatePost(1, 'hello')).toBeNull()
    expect(validatePost(16, 'hello')).toBeNull()
  })
})

function installBridge(postOk: boolean) {
  const postText = vi.fn(async () =>
    postOk
      ? { ok: true as const, value: { id: 'block-1' } }
      : { ok: false as const, error: 'rejected' },
  )
  ;(globalThis as { window?: unknown }).window = {
    bpDesktop: { node: { postText }, lifecycle: {}, feed: {}, versions: () => ({}) },
  }
  return { postText }
}

afterEach(() => {
  delete (globalThis as { window?: unknown }).window
})

describe('useCompose', () => {
  it('does not call the node when validation fails', async () => {
    const { postText } = installBridge(true)
    const c = useCompose()
    const id = await c.post(1, '')
    expect(id).toBeNull()
    expect(postText).not.toHaveBeenCalled()
    expect(c.error.value).toBeTruthy()
  })

  it('submits the intent and returns the block id', async () => {
    const { postText } = installBridge(true)
    const c = useCompose()
    const id = await c.post(2, '  hello  ')
    expect(id).toBe('block-1')
    expect(postText).toHaveBeenCalledWith({ publicAudience: 2, text: 'hello' })
    expect(c.error.value).toBeNull()
  })

  it('surfaces the node error on failure', async () => {
    installBridge(false)
    const c = useCompose()
    const id = await c.post(1, 'hi')
    expect(id).toBeNull()
    expect(c.error.value).toBe('rejected')
  })
})
