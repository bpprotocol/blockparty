import { describe, expect, it } from 'vitest'
import { avatarGradient, hashSeed, initials, shortHex, timeAgo } from './useFormat'

describe('shortHex', () => {
  it('returns a dash for empty input', () => {
    expect(shortHex('')).toBe('—')
  })

  it('leaves short values untouched', () => {
    expect(shortHex('abcd')).toBe('abcd')
  })

  it('elides the middle of long values', () => {
    expect(shortHex('0123456789abcdef')).toBe('012345…cdef')
  })
})

describe('initials', () => {
  it('uppercases the first two characters', () => {
    expect(initials('deadbeef')).toBe('DE')
  })

  it('falls back for an empty seed', () => {
    expect(initials('')).toBe('??')
  })
})

describe('avatarGradient', () => {
  it('is deterministic per seed', () => {
    expect(avatarGradient('peer-a')).toBe(avatarGradient('peer-a'))
  })

  it('differs across seeds', () => {
    expect(avatarGradient('peer-a')).not.toBe(avatarGradient('peer-b'))
  })

  it('hashes to a 32-bit unsigned value', () => {
    expect(hashSeed('peer-a')).toBeGreaterThanOrEqual(0)
    expect(hashSeed('peer-a')).toBeLessThan(2 ** 32)
  })
})

describe('timeAgo', () => {
  const now = 1_700_000_000_000 // ms
  const at = (secsAgo: number) => timeAgo(now / 1000 - secsAgo, now)

  it('renders nothing for a zero timestamp', () => {
    expect(timeAgo(0, now)).toBe('')
  })

  it('renders fresh posts as just now', () => {
    expect(at(5)).toBe('just now')
  })

  it('renders minutes, hours and days', () => {
    expect(at(120)).toBe('2m ago')
    expect(at(3 * 3600)).toBe('3h ago')
    expect(at(2 * 86400)).toBe('2d ago')
  })

  it('falls back to a date past a week', () => {
    expect(at(20 * 86400)).toBe(new Date(now - 20 * 86400 * 1000).toLocaleDateString())
  })
})
