import { describe, expect, it } from 'vitest'
import { createRouterTransport, type ServiceImpl, type Transport } from '@connectrpc/connect'
import { NodeService } from './gen/node_pb'
import { NodeClient } from './node-client'

// fakeNode builds an in-memory transport serving NodeService with the given
// handlers — no real node or network needed.
function fakeNode(handlers: Partial<ServiceImpl<typeof NodeService>>): Transport {
  return createRouterTransport(({ service }) => {
    service(NodeService, handlers as ServiceImpl<typeof NodeService>)
  })
}

describe('NodeClient', () => {
  it('maps GetStatus to a plain DTO (bigint → number)', async () => {
    const client = new NodeClient(
      fakeNode({
        getStatus: () => ({
          version: '0.1.0',
          mode: 'personal',
          worldLoaded: true,
          world: 'abc123',
          identity: 'id123',
          blockCount: 7n,
          canAuthor: true,
        }),
      }),
    )
    const r = await client.getStatus()
    expect(r.ok).toBe(true)
    if (r.ok) {
      expect(r.value.blockCount).toBe(7)
      expect(r.value.mode).toBe('personal')
      expect(r.value.worldLoaded).toBe(true)
    }
  })

  it('returns ok:false when the node errors (connection problem)', async () => {
    const client = new NodeClient(
      fakeNode({
        getStatus: () => {
          throw new Error('node unavailable')
        },
      }),
    )
    const r = await client.getStatus()
    expect(r.ok).toBe(false)
    if (!r.ok) expect(r.error.length).toBeGreaterThan(0)
  })

  it('forwards post intent and returns the block id', async () => {
    let seen: { publicAudience: number; text: string } | undefined
    const client = new NodeClient(
      fakeNode({
        postText: (req: { publicAudience: number; text: string }) => {
          seen = { publicAudience: req.publicAudience, text: req.text }
          return { id: 'block-abc' }
        },
      }),
    )
    const r = await client.postText({ publicAudience: 2, text: 'hello' })
    expect(r.ok && r.value.id).toBe('block-abc')
    expect(seen?.publicAudience).toBe(2)
    expect(seen?.text).toBe('hello')
  })
})
