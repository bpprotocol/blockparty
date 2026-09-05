// electron-builder beforePack hook (#48): build the bundled bpnode for the
// target being packaged, so each artifact carries a node for its own platform
// and architecture. The client-only config (#51) does not use this hook — that
// build deliberately ships no node.

import { buildNode } from './build-node.mjs'

// electron-builder passes Arch as an enum; map it back to the names Go needs.
const ARCH_NAMES = ['ia32', 'x64', 'armv7l', 'arm64', 'universal']

export function archName(arch) {
  if (typeof arch === 'string') return arch
  const name = ARCH_NAMES[arch]
  if (!name) throw new Error(`unknown electron-builder arch: ${String(arch)}`)
  return name
}

export default async function beforePack(context) {
  const platform = context.electronPlatformName
  let arch = archName(context.arch)
  // A universal macOS app embeds both slices; build the arm64 node, which
  // Rosetta cannot help with, and let x64 hosts run it natively on Apple
  // silicon. (Universal is not a default target here.)
  if (arch === 'universal') arch = 'arm64'
  buildNode({ platform, arch })
}
