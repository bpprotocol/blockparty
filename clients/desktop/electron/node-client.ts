import { createClient, type Client, type Interceptor, type Transport } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-node'
import { NodeService } from './gen/node_pb'
import type {
  BlockSummary,
  BootstrapRequest,
  ListFilter,
  NodeApi,
  NodeStatus,
  PostTextRequest,
  Result,
} from './bridge'

// bearerInterceptor attaches the node API token to every request (the #29 trust
// boundary). The token is held only in the main process.
export function bearerInterceptor(token: string): Interceptor {
  return (next) => (req) => {
    req.header.set('Authorization', `Bearer ${token}`)
    return next(req)
  }
}

// nodeTransport builds an HTTP/1.1 Connect transport to the node API.
export function nodeTransport(baseUrl: string, token: string): Transport {
  return createConnectTransport({
    baseUrl,
    httpVersion: '1.1',
    interceptors: [bearerInterceptor(token)],
  })
}

// NodeClient wraps the generated Connect client, converting wire messages to the
// plain DTOs the renderer consumes and turning errors into Result envelopes.
export class NodeClient implements NodeApi {
  private readonly client: Client<typeof NodeService>

  constructor(transport: Transport) {
    this.client = createClient(NodeService, transport)
  }

  async getStatus(): Promise<Result<NodeStatus>> {
    return wrap(async () => {
      const r = await this.client.getStatus({})
      return {
        version: r.version,
        mode: r.mode,
        worldLoaded: r.worldLoaded,
        world: r.world,
        identity: r.identity,
        blockCount: Number(r.blockCount),
        canAuthor: r.canAuthor,
      }
    })
  }

  async bootstrapWorld(
    req: BootstrapRequest,
  ): Promise<Result<{ world: string; identity: string }>> {
    return wrap(async () => {
      const r = await this.client.bootstrapWorld(req)
      return { world: r.world, identity: r.identity }
    })
  }

  async postText(req: PostTextRequest): Promise<Result<{ id: string }>> {
    return wrap(async () => {
      const r = await this.client.postText({ publicAudience: req.publicAudience, text: req.text })
      return { id: r.id }
    })
  }

  async getBlock(id: string): Promise<Result<import('./bridge').BlockView>> {
    return wrap(async () => {
      const r = await this.client.getBlock({ id })
      return {
        summary: summaryDTO(r.summary),
        decrypted: r.decrypted,
        text: r.text,
      }
    })
  }

  async listBlocks(filter: ListFilter): Promise<Result<BlockSummary[]>> {
    return wrap(async () => {
      const r = await this.client.listBlocks({
        audienceCode: filter.audienceCode ?? '',
        typeCode: filter.typeCode ?? '',
        author: filter.author ?? '',
        from: BigInt(filter.from ?? 0),
        to: BigInt(filter.to ?? 0),
      })
      return r.blocks.map(summaryDTO)
    })
  }
}

function summaryDTO(s: { [k: string]: unknown } | undefined): BlockSummary {
  return {
    id: str(s?.id),
    audienceCode: str(s?.audienceCode),
    typeCode: str(s?.typeCode),
    author: str(s?.author),
    timestamp: Number((s?.timestamp as bigint) ?? 0n),
    receivedAt: Number((s?.receivedAt as bigint) ?? 0n),
  }
}

function str(v: unknown): string {
  return typeof v === 'string' ? v : ''
}

async function wrap<T>(fn: () => Promise<T>): Promise<Result<T>> {
  try {
    return { ok: true, value: await fn() }
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) }
  }
}
