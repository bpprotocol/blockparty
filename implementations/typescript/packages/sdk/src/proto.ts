// Protobuf-es runtime helpers, re-exported so consumers can build, encode, and
// decode payload messages (whose generated types/schemas the SDK also exports)
// without depending on @bufbuild/protobuf directly.
export { create, toBinary, fromBinary } from "@bufbuild/protobuf";
export type { Message, DescMessage } from "@bufbuild/protobuf";
