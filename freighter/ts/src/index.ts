// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export type { ErrorPayload, MatchableErrorType, TypedError } from "@/errors";
export {
  BaseTypedError,
  decodeError,
  encodeError,
  EOF,
  errorMatcher,
  errorZ,
  registerError,
  StreamClosed,
  Unreachable,
} from "@/errors";
export { HTTPClient } from "@/http";
export type { Context, Middleware, Next } from "@/middleware";
export type { Stream, StreamClient } from "@/stream";
export type { UnaryClient } from "@/unary";
export { sendRequired, unaryWithBreaker } from "@/unary";
export { WebSocketClient } from "@/websocket";
