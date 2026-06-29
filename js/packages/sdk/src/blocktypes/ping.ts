import { create } from "@bufbuild/protobuf";
import {
  RpcRequestSchema,
  RpcResponseSchema,
  type RpcRequest,
  type RpcResponse,
} from "../blockpb/block_types_pb.js";

/** The canonical method identifier for the core ping RPC. */
export const METHOD_CORE_PING = "bpprotocol.org/v1/rpc/core.ping";

export class NotPingError extends Error {
  constructor() {
    super("blocktypes: not a core.ping");
    this.name = "NotPingError";
  }
}

/** Build a core.ping rpc.request carrying the sender's timestamp (unix seconds). */
export function buildPing(id: string, timestamp: number): RpcRequest {
  return create(RpcRequestSchema, {
    id,
    method: METHOD_CORE_PING,
    params: { timestamp: String(timestamp) },
  });
}

/**
 * Build the rpc.response for a core.ping request, echoing the original timestamp
 * and recording when it was received, the responder's clock, and a node string.
 * Throws NotPingError if req is not a core.ping.
 */
export function handlePing(
  req: RpcRequest,
  receivedAt: number,
  serverTime: number,
  node: string,
): RpcResponse {
  if (req.method !== METHOD_CORE_PING) throw new NotPingError();
  return create(RpcResponseSchema, {
    id: req.id,
    result: {
      timestamp: req.params["timestamp"] ?? "",
      received_at: String(receivedAt),
      server_time: String(serverTime),
      node,
    },
  });
}

/** The parsed payload of a core.ping rpc.response. */
export interface PingResult {
  timestamp: number;
  receivedAt: number;
  serverTime: number;
  node: string;
}

/** Parse a core.ping rpc.response. */
export function parsePingResponse(resp: RpcResponse): PingResult {
  return {
    timestamp: Number(resp.result["timestamp"] ?? "0"),
    receivedAt: Number(resp.result["received_at"] ?? "0"),
    serverTime: Number(resp.result["server_time"] ?? "0"),
    node: resp.result["node"] ?? "",
  };
}
