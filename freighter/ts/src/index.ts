// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export { WebSocketClient } from "./websocket";
export {
  MsgpackEncoderDecoder,
  JSONEncoderDecoder,
  registerCustomTypeEncoder,
  ENCODERS,
} from "./encoder";
export type { EncoderDecoder } from "./encoder";
export type { StreamClient, Stream } from "./stream";
export type { UnaryClient } from "./unary";
export { HTTPClientFactory } from "./http";
export { default as URL } from "./url";
export type { TypedError, ErrorPayload } from "./errors";
export {
  encodeError,
  decodeError,
  registerError,
  BaseTypedError,
  ErrorPayloadSchema,
  EOF,
  StreamClosed,
  Unreachable,
} from "./errors";
export type { Middleware, Next, MetaData } from "./middleware";
export { logMiddleware } from "./util/log";
