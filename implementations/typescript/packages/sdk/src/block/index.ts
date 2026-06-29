// Block envelope (issue #14): canonical serialization, the two-layer signature
// model, and co-signatures, mirroring the Go reference in ../../../../sdk/block.
// Signatures are over a domain-separated, length-prefixed preimage of the
// signed fields (not the wire bytes), shared verbatim with Go so blocks
// interoperate. The generated wire types live in ../blockpb.
export * from "./block.js";
