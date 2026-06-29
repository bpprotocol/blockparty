import { mkdirSync, writeFileSync, readdirSync, readFileSync } from "node:fs";
import { join, extname } from "node:path";
import { encode, decode, idHex, type Block } from "@blockparty/sdk";

/** File extension for serialized blocks on the filesystem transport. */
export const BLOCK_EXT = ".block";

/**
 * A filesystem ("sneakernet") transport: blocks are written as individual files
 * named by their hex ID and read back by scanning a directory.
 */
export class FileStore {
  constructor(readonly dir: string) {}

  /** Serialize a block and write it as <id>.block; returns the file path. */
  write(b: Block): string {
    mkdirSync(this.dir, { recursive: true });
    const path = join(this.dir, idHex(b) + BLOCK_EXT);
    writeFileSync(path, encode(b));
    return path;
  }

  /** Read and decode every *.block file in the directory (missing dir → none). */
  readAll(): Block[] {
    let entries: string[];
    try {
      entries = readdirSync(this.dir);
    } catch {
      return [];
    }
    const blocks: Block[] = [];
    for (const e of entries) {
      if (extname(e) !== BLOCK_EXT) continue;
      blocks.push(decode(readFileSync(join(this.dir, e))));
    }
    return blocks;
  }
}
