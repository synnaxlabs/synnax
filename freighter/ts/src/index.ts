// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export {
  ENCODERS,
  JSONEncoderDecoder,
  MsgpackEncoderDecoder,
  registerCustomTypeEncoder,
} from "@/encoder";
export type { EncoderDecoder } from "@/encoder";
export {
  BaseTypedError,
  decodeError,
  encodeError,
  EOF,
  errorZ,
  registerError,
  StreamClosed,
  Unreachable,
} from "@/errors";
export type { ErrorZ as ErrorPayload, TypedError } from "@/errors";
export { HTTPClientFactory } from "@/http";
export type { Context as MetaData, Middleware, Next } from "@/middleware";
export type { Stream, StreamClient } from "@/stream";
export type { UnaryClient } from "@/unary";
export { WebSocketClient } from "@/websocket";
