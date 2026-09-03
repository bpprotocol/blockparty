// Presentation helpers shared by the social-styled views: short hex labels,
// relative timestamps and deterministic avatar colours. All pure, so they are
// unit-tested and safe to call from render functions.

// shortHex trims a long hex identifier (address, block ID) to a readable stub.
export function shortHex(s: string, head = 6, tail = 4): string {
  if (!s) return '—'
  if (s.length <= head + tail + 1) return s
  return `${s.slice(0, head)}…${s.slice(-tail)}`
}

// initials takes the first two hex characters as an avatar monogram. Addresses
// are hex, so this is stable and collision-tolerant (the colour disambiguates).
export function initials(seed: string): string {
  return (seed || '??').slice(0, 2).toUpperCase()
}

// hashSeed is a small FNV-1a over the seed, used to pick avatar hues.
export function hashSeed(seed: string): number {
  let h = 0x811c9dc5
  for (let i = 0; i < seed.length; i++) {
    h ^= seed.charCodeAt(i)
    h = Math.imul(h, 0x01000193) >>> 0
  }
  return h
}

// avatarGradient maps a seed to a stable two-stop gradient, so every peer keeps
// the same "profile picture" across sessions without any avatar storage.
export function avatarGradient(seed: string): string {
  const h = hashSeed(seed || 'anon')
  const hue = h % 360
  const hue2 = (hue + 38) % 360
  return `linear-gradient(135deg, hsl(${hue} 62% 56%), hsl(${hue2} 68% 46%))`
}

// timeAgo renders a unix-seconds timestamp the way a feed does: seconds and
// minutes for fresh posts, then hours, days, and finally an absolute date.
export function timeAgo(ts: number, now: number = Date.now()): string {
  if (!ts) return ''
  const secs = Math.max(0, Math.floor(now / 1000 - ts))
  if (secs < 45) return 'just now'
  const mins = Math.floor(secs / 60)
  if (mins < 60) return `${Math.max(1, mins)}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`
  return new Date(ts * 1000).toLocaleDateString()
}
