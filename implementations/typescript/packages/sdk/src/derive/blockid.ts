import { hmacSha256, keccak256, sha256 } from "../crypto/index.js";
import { bytesToHex, utf8ToBytes } from "../bytes.js";

/**
 * Derive the content-binding block identifier:
 *
 *   dataHash = hex(keccak256(data))
 *   message  = "block.v"+version+":"+timestamp+"/"+address+"/"+typeCodeHex+"/"+dataHash
 *   id       = hex(sha256(hmacSha256(key = hex(audienceCode), message)))
 *
 * Including the payload hash makes the ID change whenever data changes.
 * `timestamp` is unix seconds; values stay well within Number's safe range.
 */
export function getBlockID(
  version: number,
  timestamp: number,
  audienceCode: Uint8Array,
  address: string,
  typeCode: Uint8Array,
  data: Uint8Array,
): string {
  const dataHash = bytesToHex(keccak256(data));
  const message =
    "block.v" +
    version +
    ":" +
    timestamp +
    "/" +
    address +
    "/" +
    bytesToHex(typeCode) +
    "/" +
    dataHash;
  const seed = hmacSha256(utf8ToBytes(bytesToHex(audienceCode)), utf8ToBytes(message));
  return bytesToHex(sha256(seed));
}
