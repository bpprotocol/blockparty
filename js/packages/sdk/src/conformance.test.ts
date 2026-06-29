import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { bytesToHex, utf8ToBytes } from "./bytes.js";
import { keccak256, makeKyberPair, makeMldsaPair } from "./crypto/index.js";
import { openWorld, getTypeCode, getAudienceCode, getBlockID } from "./derive/index.js";
import { openIdentity } from "./identity/index.js";
import { publicAudienceSecret, inboxAudience } from "./audiences/index.js";
import { contentKey, aad, sealWithNonce } from "./encryption/index.js";
import { newBlock, signBlock, encode, idHex } from "./block/index.js";
import { connectionSecret0, rotatedSecret, audienceCode } from "./connections/index.js";

// Fixed inputs — identical to the Go vectors generator (sdk/vectors/vectors.go).
const WorldSeed = "bpprotocol.org/v1/test-world";
const Passphrase = "correct horse battery staple";
const KeySeed = "bpprotocol.org/v1/test-identity";
const TypeURN = "bpprotocol.org/v1/types/content.post";
const AudienceURN = "bpprotocol.org/v1/audience/public-1";
const AEADSecretSeed = "bpprotocol.org/v1/test-secret";
const AEADNonce = "bpprotocol/v1/test-nonce";
const Timestamp = 1700000000;
const Version = 1;

const hex = (b: Uint8Array) => bytesToHex(b);
const fill = (n: number, byte: number) => new Uint8Array(n).fill(byte);

/** Re-derive the full conformance vector set from the SDK alone. */
function computeVectors() {
  const w = openWorld(WorldSeed);
  const typeCode = getTypeCode(w, TypeURN);
  const audCode = getAudienceCode(w, AudienceURN);
  const id = openIdentity(w, Passphrase);

  const kyber = makeKyberPair(utf8ToBytes(KeySeed));
  const mldsa = makeMldsaPair(utf8ToBytes(KeySeed));

  const aeadSecret = keccak256(utf8ToBytes(AEADSecretSeed));
  const nonce = utf8ToBytes(AEADNonce);
  const aadBytes = aad(Version, typeCode, audCode, Timestamp);
  const aeadData = sealWithNonce(
    aeadSecret,
    audCode,
    nonce,
    utf8ToBytes("hello, audience"),
    aadBytes,
  );

  const ss1 = fill(32, 0x11);
  const ss2 = fill(32, 0x22);
  const cNonce = fill(32, 0x33);
  const nonceResp = fill(32, 0x44);
  const ss3 = fill(32, 0x55);
  const ct3 = fill(32, 0x66);
  const cSecret0 = connectionSecret0(ss1, ss2, cNonce, nonceResp);
  const cSecret1 = rotatedSecret(cSecret0, ss3, ct3, 1);

  const blk = newBlock(
    id.address,
    typeCode,
    audCode,
    Timestamp,
    utf8ToBytes("block fixture payload"),
  );
  signBlock(blk, w, id.mldsa);
  const encoded = encode(blk);

  return {
    crypto: {
      seed: KeySeed,
      kyberPubKeccak: hex(keccak256(kyber.publicKey)),
      mldsaPubKeccak: hex(keccak256(mldsa.publicKey)),
    },
    world: {
      seed: WorldSeed,
      walletSalt: hex(w.walletSalt),
      typeSalt: hex(w.typeSalt),
      audienceSalt: hex(w.audienceSalt),
      signingKeyKeccak: hex(keccak256(w.signingKey.publicKey)),
      typeCode: hex(typeCode),
      audienceCode: hex(audCode),
    },
    identity: {
      world: WorldSeed,
      passphrase: Passphrase,
      address: id.address,
      mldsaPubKeccak: hex(keccak256(id.mldsa.publicKey)),
      kyberPubKeccak: hex(keccak256(id.kyber.publicKey)),
    },
    blockID: {
      version: Version,
      timestamp: Timestamp,
      data: "hello",
      id: getBlockID(Version, Timestamp, audCode, id.address, typeCode, utf8ToBytes("hello")),
    },
    audiences: {
      publicAudienceSecret: hex(publicAudienceSecret(w, AudienceURN)),
      inboxCode: hex(inboxAudience(w, id.address).code),
    },
    aead: {
      secretSeed: AEADSecretSeed,
      nonce: AEADNonce,
      plaintext: "hello, audience",
      contentKey: hex(contentKey(aeadSecret, audCode, nonce)),
      aad: hex(aadBytes),
      data: hex(aeadData),
    },
    block: {
      passphrase: Passphrase,
      typeURN: TypeURN,
      audienceURN: AudienceURN,
      timestamp: Timestamp,
      data: "block fixture payload",
      idHex: idHex(blk),
      encodedKeccak: hex(keccak256(encoded)),
    },
    connection: {
      ss1: hex(ss1),
      ss2: hex(ss2),
      nonce: hex(cNonce),
      nonceResponse: hex(nonceResp),
      ss3: hex(ss3),
      ct3: hex(ct3),
      connectionSecret0: hex(cSecret0),
      audienceCode0: hex(audienceCode(cSecret0, 0)),
      audienceCode1: hex(audienceCode(cSecret1, 1)),
    },
    burn: {
      passphrase: Passphrase,
      identity: id.address,
      mldsaSkKeccak: hex(keccak256(id.mldsa.secretKey)),
      kyberSkKeccak: hex(keccak256(id.kyber.secretKey)),
      burnNotice: "voluntary",
    },
  };
}

describe("cross-client conformance", () => {
  it("@blockparty/sdk reproduces the entire shared vectors.json", () => {
    const expected = JSON.parse(
      readFileSync(
        fileURLToPath(new URL("../../../../sdk/vectors/vectors.json", import.meta.url)),
        "utf8",
      ),
    );
    expect(computeVectors()).toEqual(expected);
  });
});
