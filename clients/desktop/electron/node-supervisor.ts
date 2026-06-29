import { spawn, type ChildProcess } from 'node:child_process'
import { readFileSync } from 'node:fs'
import path from 'node:path'
import type { LifecycleMode, LifecycleState, NodeState } from './bridge'

export interface SupervisorOptions {
  mode: LifecycleMode
  baseUrl: string // http(s)://host:port the API is reached on
  apiAddr: string // host:port the managed node binds
  dataDir: string // node data dir (holds api.token)
  binPath: string // path to the bpnode binary (managed mode)
  nodeMode: string // 'personal' | 'relay'
  readyTimeoutMs?: number
  stopTimeoutMs?: number
  maxRestarts?: number
}

export interface NodeConnection {
  baseUrl: string
  token: string
}

const LOG_RING = 500

// NodeSupervisor manages the local node process: in managed mode it spawns and
// supervises a bundled bpnode (start/stop/restart, crash recovery, log capture);
// in attach mode it skips spawning and connects to an externally-run daemon.
export class NodeSupervisor {
  private readonly opts: Required<SupervisorOptions>
  private child?: ChildProcess
  private state: NodeState = 'stopped'
  private shuttingDown = false
  private restarts = 0
  private readonly logs: string[] = []

  constructor(opts: SupervisorOptions) {
    this.opts = {
      readyTimeoutMs: 15_000,
      stopTimeoutMs: 8_000,
      maxRestarts: 5,
      ...opts,
    }
  }

  // start brings the node up and returns how to reach it. In attach mode it does
  // not spawn anything.
  async start(): Promise<NodeConnection> {
    if (this.opts.mode === 'attach') {
      this.state = 'running'
      this.log(`attaching to external node at ${this.opts.baseUrl}`)
      return { baseUrl: this.opts.baseUrl, token: this.readToken(true) }
    }
    await this.spawnAndWait()
    return { baseUrl: this.opts.baseUrl, token: this.readToken(false) }
  }

  // stop terminates a managed node gracefully (SIGTERM, then SIGKILL on timeout).
  async stop(): Promise<void> {
    this.shuttingDown = true
    const child = this.child
    if (!child || child.exitCode !== null || child.signalCode !== null) {
      this.state = 'stopped'
      return
    }
    const exited = new Promise<void>((resolve) => child.once('exit', () => resolve()))
    child.kill('SIGTERM')
    const clean = await Promise.race([
      exited.then(() => true),
      delay(this.opts.stopTimeoutMs).then(() => false),
    ])
    if (!clean) {
      this.log('node did not stop in time; sending SIGKILL')
      child.kill('SIGKILL')
      await exited
    }
    this.state = 'stopped'
    this.log('node stopped')
  }

  getState(): LifecycleState {
    return {
      mode: this.opts.mode,
      state: this.state,
      endpoint: this.opts.baseUrl,
      restarts: this.restarts,
    }
  }

  recentLogs(): string[] {
    return [...this.logs]
  }

  private async spawnAndWait(): Promise<void> {
    this.spawn()
    await this.waitReady()
    this.state = 'running'
    this.log(`node ready at ${this.opts.baseUrl}`)
  }

  private spawn(): void {
    this.state = 'starting'
    const args = [
      '--mode',
      this.opts.nodeMode,
      '--data-dir',
      this.opts.dataDir,
      '--api-addr',
      this.opts.apiAddr,
      '--log-format',
      'json',
    ]
    this.log(`spawning ${this.opts.binPath} ${args.join(' ')}`)
    const child = spawn(this.opts.binPath, args, { stdio: ['ignore', 'pipe', 'pipe'] })
    child.stdout?.on('data', (d: Buffer) => this.capture(d))
    child.stderr?.on('data', (d: Buffer) => this.capture(d))
    child.on('exit', (code, signal) => this.onExit(code, signal))
    child.on('error', (err) => this.log(`spawn error: ${err.message}`))
    this.child = child
  }

  private onExit(code: number | null, signal: NodeJS.Signals | null): void {
    this.log(`node exited (code=${code ?? '-'} signal=${signal ?? '-'})`)
    if (this.shuttingDown) {
      this.state = 'stopped'
      return
    }
    this.state = 'crashed'
    if (this.restarts >= this.opts.maxRestarts) {
      this.log('max restarts reached; not restarting')
      return
    }
    this.restarts++
    const backoff = Math.min(500 * 2 ** (this.restarts - 1), 10_000)
    this.log(`restarting in ${backoff}ms (attempt ${this.restarts}/${this.opts.maxRestarts})`)
    setTimeout(() => {
      if (!this.shuttingDown) {
        this.spawnAndWait().catch((e: unknown) => this.log(`restart failed: ${String(e)}`))
      }
    }, backoff)
  }

  private async waitReady(): Promise<void> {
    const deadline = Date.now() + this.opts.readyTimeoutMs
    while (Date.now() < deadline) {
      try {
        const res = await fetch(`${this.opts.baseUrl}/healthz`, {
          signal: AbortSignal.timeout(1000),
        })
        if (res.ok) return
      } catch {
        // not up yet
      }
      await delay(150)
    }
    throw new Error(`node did not become ready within ${this.opts.readyTimeoutMs}ms`)
  }

  private readToken(optional: boolean): string {
    try {
      return readFileSync(path.join(this.opts.dataDir, 'api.token'), 'utf8').trim()
    } catch (e) {
      if (optional) return ''
      throw new Error(`could not read node API token: ${String(e)}`)
    }
  }

  private capture(buf: Buffer): void {
    for (const line of buf.toString('utf8').split('\n')) {
      if (line.trim()) this.log(line.trim())
    }
  }

  private log(line: string): void {
    this.logs.push(line)
    if (this.logs.length > LOG_RING) this.logs.shift()
  }
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}
