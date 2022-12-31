// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export { WebSocketClient } from "./lib/websocket";
export {
  MsgpackEncoderDecoder,
  JSONEncoderDecoder,
  registerCustomTypeEncoder,
  ENCODERS,
} from "./lib/encoder";
export type { EncoderDecoder } from "./lib/encoder";
export type { StreamClient, Stream } from "./lib/stream";
export type { UnaryClient } from "./lib/unary";
export { HTTPClientFactory } from "./lib/http";
export { default as URL } from "./lib/url";
export type { TypedError, ErrorPayload } from "./lib/errors";
export {
  encodeError,
  decodeError,
  registerError,
  BaseTypedError,
  ErrorPayloadSchema,
  EOF,
  StreamClosed,
  Unreachable,
} from "./lib/errors";
export type { Middleware, Next, MetaData } from "./lib/middleware";
export { logMiddleware } from "./lib/util/log";
