import { afterEach, describe, expect, it, vi } from 'vitest'
import type { IdentityCard } from '../electron/bridge'
import { formatCard, parseCard, useConnections } from './useConnections'

const card: IdentityCard = { address: 'addr1', kyberPub: 'kpub', mldsaPub: 'mpub' }

describe('card encoding', () => {
  it('round-trips a card', () => {
    expect(parseCard(formatCard(card))).toEqual(card)
  })
  it('rejects malformed cards', () => {
    expect(parseCard('')).toBeNull()
    expect(parseCard('only.two')).toBeNull()
    expect(parseCard('a..c')).toBeNull()
    expect(parseCard('  addr1.kpub.mpub  ')).toEqual(card)
  })
})

function installBridge(over: { addOk?: boolean; startOk?: boolean } = {}) {
  const addPeer = vi.fn(async () =>
    over.addOk === false
      ? { ok: false as const, error: 'bad key' }
      : { ok: true as const, value: undefined },
  )
  const start = vi.fn(async () =>
    over.startOk === false
      ? { ok: false as const, error: 'no peer' }
      : { ok: true as const, value: { requestId: 'r1' } },
  )
  const list = vi.fn(async () => ({ ok: true as const, value: [] }))
  ;(globalThis as { window?: unknown }).window = {
    bpDesktop: {
      connections: {
        addPeer,
        start,
        list,
        getIdentity: vi.fn(),
        rotate: vi.fn(),
        close: vi.fn(),
        sendText: vi.fn(),
        messages: vi.fn(),
      },
    },
  }
  return { addPeer, start }
}

afterEach(() => {
  delete (globalThis as { window?: unknown }).window
})

describe('addAndConnect', () => {
  it('rejects an unparseable card before calling the node', async () => {
    const { addPeer } = installBridge()
    const c = useConnections()
    expect(await c.addAndConnect('garbage')).toBeNull()
    expect(addPeer).not.toHaveBeenCalled()
    expect(c.error.value).toBeTruthy()
  })

  it('adds the peer then starts the handshake', async () => {
    const { addPeer, start } = installBridge()
    const c = useConnections()
    const addr = await c.addAndConnect(formatCard(card))
    expect(addr).toBe('addr1')
    expect(addPeer).toHaveBeenCalledWith(card)
    expect(start).toHaveBeenCalledWith('addr1')
  })

  it('surfaces an addPeer error without starting', async () => {
    const { start } = installBridge({ addOk: false })
    const c = useConnections()
    expect(await c.addAndConnect(formatCard(card))).toBeNull()
    expect(start).not.toHaveBeenCalled()
    expect(c.error.value).toBe('bad key')
  })
})
