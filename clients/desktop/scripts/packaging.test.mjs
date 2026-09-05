import { describe, expect, it } from 'vitest'
import { goEnvFor, hostTarget } from './build-node.mjs'
import { archName } from './before-pack.mjs'

describe('goEnvFor', () => {
  it('maps every packaged platform to a Go target', () => {
    expect(goEnvFor('linux', 'x64')).toEqual({
      GOOS: 'linux',
      GOARCH: 'amd64',
      binName: 'bpnode',
    })
    expect(goEnvFor('darwin', 'arm64')).toEqual({
      GOOS: 'darwin',
      GOARCH: 'arm64',
      binName: 'bpnode',
    })
    // Windows is the one target whose binary name differs — the same rule the
    // supervisor uses to find it (node-config.ts).
    expect(goEnvFor('win32', 'x64')).toEqual({
      GOOS: 'windows',
      GOARCH: 'amd64',
      binName: 'bpnode.exe',
    })
  })

  it('rejects targets it cannot build for, rather than shipping no node', () => {
    expect(() => goEnvFor('sunos', 'x64')).toThrow(/unsupported platform/)
    expect(() => goEnvFor('linux', 'mips')).toThrow(/unsupported architecture/)
  })

  it('defaults to this machine', () => {
    const t = hostTarget()
    expect(() => goEnvFor(t.platform, t.arch)).not.toThrow()
  })
})

describe('archName', () => {
  it('translates electron-builder Arch enum values', () => {
    expect(archName(0)).toBe('ia32')
    expect(archName(1)).toBe('x64')
    expect(archName(2)).toBe('armv7l')
    expect(archName(3)).toBe('arm64')
    expect(archName('arm64')).toBe('arm64')
  })

  it('throws on an arch it does not know', () => {
    expect(() => archName(42)).toThrow(/unknown electron-builder arch/)
  })
})
