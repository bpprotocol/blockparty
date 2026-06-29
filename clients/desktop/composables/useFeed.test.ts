import { describe, expect, it } from 'vitest'
import { mergeItem, type FeedItem } from './useFeed'

function item(id: string, timestamp: number, text = 't'): FeedItem {
  return { id, author: 'a', timestamp, text }
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
