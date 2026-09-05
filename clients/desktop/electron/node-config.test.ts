import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { chmodSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { hasBundledNode, resolveSupervisorOptions } from './node-config'

const ENV_KEYS = [
  'BPNODE_ATTACH',
  'BPNODE_BIN',
  'BPNODE_API_ADDR',
  'BPNODE_DATA_DIR',
  'BPNODE_MODE',
] as const

let saved: Record<string, string | undefined>
let dirs: string[] = []

// resources builds a fake packaged resources dir, optionally holding a runnable
// bpnode for the current platform.
function resources(
  withBinary: boolean,
  name = process.platform === 'win32' ? 'bpnode.exe' : 'bpnode',
): string {
  const dir = mkdtempSync(path.join(tmpdir(), 'bpres-'))
  dirs.push(dir)
  if (withBinary) {
    const p = path.join(dir, name)
    writeFileSync(p, '#!/bin/sh\nexit 0\n')
    chmodSync(p, 0o755)
  }
  return dir
}

beforeEach(() => {
  saved = Object.fromEntries(ENV_KEYS.map((k) => [k, process.env[k]]))
  for (const k of ENV_KEYS) delete process.env[k]
})

afterEach(() => {
  for (const [k, v] of Object.entries(saved)) {
    if (v === undefined) delete process.env[k]
    else process.env[k] = v
  }
  for (const d of dirs) rmSync(d, { recursive: true, force: true })
  dirs = []
})

describe('hasBundledNode', () => {
  it('is false for a missing path and for a directory', () => {
    const dir = resources(false)
    expect(hasBundledNode(path.join(dir, 'nope'))).toBe(false)
    mkdirSync(path.join(dir, 'sub'))
    expect(hasBundledNode(path.join(dir, 'sub'))).toBe(false)
  })

  it('is true for a runnable binary', () => {
    const dir = resources(true)
    const name = process.platform === 'win32' ? 'bpnode.exe' : 'bpnode'
    expect(hasBundledNode(path.join(dir, name))).toBe(true)
  })

  it('is false for a non-executable file on POSIX', () => {
    if (process.platform === 'win32') return
    const dir = resources(false)
    const p = path.join(dir, 'bpnode')
    writeFileSync(p, 'not runnable')
    chmodSync(p, 0o644)
    expect(hasBundledNode(p)).toBe(false)
  })
})

describe('resolveSupervisorOptions', () => {
  it('manages the bundled node when one ships with the build', () => {
    const opts = resolveSupervisorOptions('/appdata', resources(true))
    expect(opts.mode).toBe('managed')
    expect(opts.canManage).toBe(true)
    expect(opts.attachConfigured).toBe(true)
    expect(opts.baseUrl).toBe('http://127.0.0.1:4400')
  })

  it('is attach-only with no endpoint when no node ships with the build', () => {
    const opts = resolveSupervisorOptions('/appdata', resources(false))
    expect(opts.mode).toBe('attach')
    expect(opts.canManage).toBe(false)
    // Nothing to attach to yet: the UI must ask for an endpoint.
    expect(opts.attachConfigured).toBe(false)
  })

  it('uses a saved external node in a client-only build', () => {
    const opts = resolveSupervisorOptions('/appdata', resources(false), {
      baseUrl: 'http://10.0.0.5:4400',
      token: 'tok',
    })
    expect(opts.mode).toBe('attach')
    expect(opts.attachConfigured).toBe(true)
    expect(opts.baseUrl).toBe('http://10.0.0.5:4400')
    expect(opts.token).toBe('tok')
  })

  it('ignores a saved external node when the build manages its own', () => {
    const opts = resolveSupervisorOptions('/appdata', resources(true), {
      baseUrl: 'http://10.0.0.5:4400',
      token: 'tok',
    })
    expect(opts.mode).toBe('managed')
    expect(opts.baseUrl).toBe('http://127.0.0.1:4400')
    expect(opts.token).toBe('')
  })

  it('honours BPNODE_ATTACH even with a bundled node', () => {
    process.env.BPNODE_ATTACH = '1'
    process.env.BPNODE_API_ADDR = '127.0.0.1:9999'
    const opts = resolveSupervisorOptions('/appdata', resources(true))
    expect(opts.mode).toBe('attach')
    // The binary is still there, so managed mode remains available.
    expect(opts.canManage).toBe(true)
    expect(opts.attachConfigured).toBe(true)
    expect(opts.baseUrl).toBe('http://127.0.0.1:9999')
  })

  it('treats a bare BPNODE_ATTACH=1 as attaching to the local default', () => {
    process.env.BPNODE_ATTACH = '1'
    const opts = resolveSupervisorOptions('/appdata', resources(false))
    expect(opts.mode).toBe('attach')
    // No endpoint given, but asking to attach means the documented dev flow
    // (a node on 127.0.0.1:4400) still connects without a setup screen.
    expect(opts.attachConfigured).toBe(true)
    expect(opts.baseUrl).toBe('http://127.0.0.1:4400')
  })

  it('honours BPNODE_BIN pointing at a node built for development', () => {
    const dir = resources(true)
    process.env.BPNODE_BIN = path.join(dir, process.platform === 'win32' ? 'bpnode.exe' : 'bpnode')
    const opts = resolveSupervisorOptions('/appdata', resources(false))
    expect(opts.mode).toBe('managed')
    expect(opts.canManage).toBe(true)
  })

  it('probes the platform-specific binary name', () => {
    const real = process.platform
    try {
      // A resources dir holding only bpnode.exe is a bundled node on Windows…
      const winOnly = resources(true, 'bpnode.exe')
      Object.defineProperty(process, 'platform', { value: 'win32', configurable: true })
      expect(resolveSupervisorOptions('/appdata', winOnly).canManage).toBe(true)

      // …and nothing at all on Linux, which looks for `bpnode`.
      Object.defineProperty(process, 'platform', { value: 'linux', configurable: true })
      expect(resolveSupervisorOptions('/appdata', winOnly).canManage).toBe(false)
    } finally {
      Object.defineProperty(process, 'platform', { value: real, configurable: true })
    }
  })
})
