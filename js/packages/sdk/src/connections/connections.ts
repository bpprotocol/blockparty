import { create } from "@bufbuild/protobuf";
import { randomBytes } from "@noble/hashes/utils.js";
import {
  ConnectRequestSchema,
  ConnectResponseSchema,
  ConnectRotateSchema,
  ConnectCloseSchema,
  type ConnectRequest,
  type ConnectResponse,
  type ConnectRotate,
  type ConnectClose,
} from "../blockpb/block_types_pb.js";
import { encapsulate, decapsulate, makeKyberPair, type KeyPair } from "../crypto/index.js";
import { codeHex } from "../derive/index.js";
import type { Identity } from "../identity/index.js";
import { audienceCode, audienceSecret, connectionSecret0, rotatedSecret } from "./derive.js";

export type { ConnectRequest, ConnectResponse, ConnectRotate, ConnectClose };

/** Length of the handshake nonces. */
export const HANDSHAKE_NONCE_SIZE = 32;

export class WrongTargetError extends Error {
  constructor() {
    super("connections: request target is not this identity");
    this.name = "WrongTargetError";
  }
}
export class RotateMismatchError extends Error {
  constructor() {
    super("connections: rotation channel mismatch");
    this.name = "RotateMismatchError";
  }
}

/**
 * A remote identity addressable for a connection: its address and static KEM
 * public key (learned from its identity block).
 */
export interface Peer {
  readonly address: string;
  readonly kyberPublicKey: Uint8Array;
}

/**
 * An established connection at a given epoch. Its audience secret and code drive
 * the private channel's encryption.
 */
export class Connection {
  #epoch: number;
  #secret: Uint8Array;
  readonly #self: Identity;
  readonly #peerAddress: string;
  readonly #peerKyber: Uint8Array;

  constructor(self: Identity, peerAddress: string, peerKyber: Uint8Array, secret: Uint8Array) {
    this.#self = self;
    this.#peerAddress = peerAddress;
    this.#peerKyber = peerKyber;
    this.#epoch = 0;
    this.#secret = secret;
  }

  get epoch(): number {
    return this.#epoch;
  }

  get peerAddress(): string {
    return this.#peerAddress;
  }

  /** The current epoch's private audience secret. */
  audienceSecret(): Uint8Array {
    return audienceSecret(this.#secret, this.#epoch);
  }

  /** The current epoch's private audience code. */
  audienceCode(): Uint8Array {
    return audienceCode(this.#secret, this.#epoch);
  }

  /**
   * Advance this connection to the next epoch with fresh KEM entropy and return
   * the connect.rotate payload (to be sent encrypted to the current audience).
   */
  rotate(): ConnectRotate {
    const { ciphertext: ct3, sharedSecret: ss3 } = encapsulate(this.#peerKyber);
    this.#secret = rotatedSecret(this.#secret, ss3, ct3, this.#epoch + 1);
    this.#epoch += 1;
    return create(ConnectRotateSchema, {
      newChannel: codeHex(this.audienceCode()),
      kemCiphertext: ct3,
    });
  }

  /**
   * Advance this connection in response to a peer's connect.rotate, verifying
   * that the recomputed audience code matches new_channel before committing.
   */
  applyRotate(rot: ConnectRotate): void {
    const ss3 = decapsulate(rot.kemCiphertext, this.#self.kyber.secretKey);
    const next = rotatedSecret(this.#secret, ss3, rot.kemCiphertext, this.#epoch + 1);
    if (codeHex(audienceCode(next, this.#epoch + 1)) !== rot.newChannel) {
      throw new RotateMismatchError();
    }
    this.#secret = next;
    this.#epoch += 1;
  }
}

/**
 * The initiator's state between sending connect.request and receiving
 * connect.response.
 */
export class PendingRequest {
  constructor(
    private readonly self: Identity,
    private readonly peer: Peer,
    private readonly eph: KeyPair,
    private readonly ss1: Uint8Array,
    private readonly nonce: Uint8Array,
  ) {}

  /**
   * Finish the initiator's handshake from the connect.response, recovering the
   * second shared secret and deriving the Connection.
   */
  complete(resp: ConnectResponse): Connection {
    const ss2 = decapsulate(resp.kemCiphertext, this.eph.secretKey);
    const secret = connectionSecret0(this.ss1, ss2, this.nonce, resp.nonceResponse);
    return new Connection(this.self, this.peer.address, this.peer.kyberPublicKey, secret);
  }
}

/**
 * Build a connect.request to peer: a fresh ephemeral KEM key for forward
 * secrecy, a shared secret encapsulated to the peer's static key, and a random
 * nonce. Complete the returned PendingRequest once the response arrives.
 */
export function startRequest(
  self: Identity,
  peer: Peer,
): { pending: PendingRequest; request: ConnectRequest } {
  const eph = makeKyberPair(randomBytes(32));
  const { ciphertext: ct1, sharedSecret: ss1 } = encapsulate(peer.kyberPublicKey);
  const nonce = randomBytes(HANDSHAKE_NONCE_SIZE);
  const request = create(ConnectRequestSchema, {
    target: peer.address,
    initKemPub: eph.publicKey,
    kemCiphertext: ct1,
    nonce,
  });
  return { pending: new PendingRequest(self, peer, eph, ss1, nonce), request };
}

/**
 * Consume a connect.request addressed to self from initiator, returning the
 * connect.response payload and the established Connection. requestBlockID is the
 * block ID of the request (carried in the response for correlation).
 */
export function acceptRequest(
  self: Identity,
  initiator: Peer,
  request: ConnectRequest,
  requestBlockID: string,
): { response: ConnectResponse; connection: Connection } {
  if (request.target !== self.address) throw new WrongTargetError();
  const ss1 = decapsulate(request.kemCiphertext, self.kyber.secretKey);
  const { ciphertext: ct2, sharedSecret: ss2 } = encapsulate(request.initKemPub);
  const nonceResponse = randomBytes(HANDSHAKE_NONCE_SIZE);
  const secret = connectionSecret0(ss1, ss2, request.nonce, nonceResponse);
  const connection = new Connection(self, initiator.address, initiator.kyberPublicKey, secret);
  const response = create(ConnectResponseSchema, {
    request: requestBlockID,
    kemCiphertext: ct2,
    nonceResponse,
  });
  return { response, connection };
}

/** Build a connect.close payload notifying peer of teardown. */
export function buildClose(peerAddress: string, reason: string): ConnectClose {
  return create(ConnectCloseSchema, { target: peerAddress, reason });
}

/** A peer's static KEM public key as a Peer, given its identity. */
export function peerFromIdentity(id: Identity): Peer {
  return { address: id.address, kyberPublicKey: id.kyber.publicKey };
}
