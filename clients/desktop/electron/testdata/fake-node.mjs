#!/usr/bin/env node
// A minimal stand-in for bpnode, used by node-supervisor.test.ts: it writes an
// api.token to the data dir and serves /healthz, so the supervisor's
// spawn → wait-ready → stop path can be tested without the real Go binary.
import http from 'node:http'
import { mkdirSync, writeFileSync } from 'node:fs'
import path from 'node:path'

const args = process.argv.slice(2)
const arg = (name) => {
  const i = args.indexOf(name)
  return i >= 0 ? args[i + 1] : undefined
}

const dataDir = arg('--data-dir') ?? '.'
const addr = arg('--api-addr') ?? '127.0.0.1:0'
mkdirSync(dataDir, { recursive: true })
writeFileSync(path.join(dataDir, 'api.token'), 'fake-token\n', { mode: 0o600 })

const [host, portStr] = addr.split(':')
const server = http.createServer((req, res) => {
  if (req.url === '/healthz') {
    res.writeHead(200, { 'content-type': 'application/json' })
    res.end('{"status":"ok"}')
  } else {
    res.writeHead(404)
    res.end()
  }
})
server.listen(Number(portStr), host, () => {
  process.stdout.write(`fake node listening on ${addr}\n`)
})
process.on('SIGTERM', () => server.close(() => process.exit(0)))
