import { afterEach, describe, expect, it, vi } from 'vitest'
import { mergeItem, useFeed, type FeedItem } from './useFeed'
import type { BlockSummary } from '../electron/bridge'

function item(id: string, timestamp: number, text = 't'): FeedItem {
  return { id, author: 'a', timestamp, text, publicAudience: 1 }
}

function summary(id: string, publicAudience: number): BlockSummary {
  return {
    id,
    audienceCode: id,
    typeCode: '',
    author: 'a',
    timestamp: 1,
    receivedAt: 1,
    publicAudience,
  }
}

describe('mergeItem', () => {
  it('keeps the list newest-first', () => {
    let list: FeedItem[] = []
    list = mergeItem(list, item('a', 100))
    list = mergeItem(list, item('b', 300))
    list = mergeItem(list, item('c', 200))
    expect(list.map((i) => i.id)).toEqual(['b', 'c', 'a'])
  })

  it('de-duplicates by id (latest text wins)', () => {
    let list = [item('a', 100, 'old')]
    list = mergeItem(list, item('a', 100, 'new'))
    expect(list).toHaveLength(1)
    expect(list[0].text).toBe('new')
  })

  it('breaks ties deterministically by id', () => {
    let list: FeedItem[] = []
    list = mergeItem(list, item('a', 100))
    list = mergeItem(list, item('b', 100))
    expect(list.map((i) => i.id)).toEqual(['b', 'a'])
  })
})

describe('useFeed.load', () => {
  afterEach(() => {
    delete (globalThis as { window?: unknown }).window
  })

  it('scopes the feed to public audiences, dropping private blocks', async () => {
    const getBlock = vi.fn(async (id: string) => ({
      ok: true as const,
      value: { decrypted: true, text: `body-${id}` },
    }))
    ;(globalThis as { window?: unknown }).window = {
      bpDesktop: {
        node: {
          listBlocks: vi.fn(async () => ({
            ok: true as const,
            value: [summary('pub', 3), summary('priv', 0)],
          })),
          getBlock,
        },
      },
    }

    const feed = useFeed()
    await feed.load()

    expect(feed.items.value.map((i) => i.id)).toEqual(['pub'])
    expect(feed.items.value[0].publicAudience).toBe(3)
    // The private block is filtered before ever fetching its payload.
    expect(getBlock).toHaveBeenCalledTimes(1)
    expect(getBlock).toHaveBeenCalledWith('pub')
  })
})
