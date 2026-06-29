import { sha512 } from "../crypto/index.js";
import { bytesToHex, concatBytes } from "../bytes.js";
import type { ChunkManifest, ContentChunkedManifest } from "../blockpb/block_types_pb.js";

export class MissingChunkError extends Error {
  constructor(id: string) {
    super(`blocktypes: missing chunk: ${id}`);
    this.name = "MissingChunkError";
  }
}

export class ChecksumError extends Error {
  constructor() {
    super("blocktypes: reassembled checksum mismatch");
    this.name = "ChecksumError";
  }
}

/**
 * Concatenate the chunk payloads named by chunkIDs (in order) and verify the
 * result against sha512Hex (the manifest's hex SHA-512). `dataByID` maps a chunk
 * block ID to that chunk's payload bytes.
 */
export function reassembleChunks(
  chunkIDs: string[],
  dataByID: Map<string, Uint8Array>,
  sha512Hex: string,
): Uint8Array {
  const parts: Uint8Array[] = [];
  for (const id of chunkIDs) {
    const d = dataByID.get(id);
    if (!d) throw new MissingChunkError(id);
    parts.push(d);
  }
  const buf = concatBytes(...parts);
  if (bytesToHex(sha512(buf)) !== sha512Hex) throw new ChecksumError();
  return buf;
}

/** Reassemble a transport-layer chunk.manifest. */
export function reassembleChunkManifest(
  manifest: ChunkManifest,
  dataByID: Map<string, Uint8Array>,
): Uint8Array {
  return reassembleChunks(manifest.chunks, dataByID, manifest.sha512);
}

/** Reassemble an application-layer content.chunked.manifest. */
export function reassembleContentChunked(
  manifest: ContentChunkedManifest,
  dataByID: Map<string, Uint8Array>,
): Uint8Array {
  return reassembleChunks(manifest.chunks, dataByID, manifest.sha512);
}
