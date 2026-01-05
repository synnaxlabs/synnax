// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors, URL } from "@synnaxlabs/x";

/**
 * Base class for all freighter-specific errors
 */
export class FreighterError extends errors.createTyped("freighter") {}

/**
 * Error thrown when reaching the end of a file or stream
 */
export class EOF extends FreighterError.sub("eof") {
  constructor() {
    super("EOF");
  }
}

/**
 * Error thrown when attempting to operate on a closed stream
 */
export class StreamClosed extends FreighterError.sub("stream_closed") {
  constructor() {
    super("StreamClosed");
  }
}

/**
 * Arguments for constructing an Unreachable error
 */
export interface UnreachableArgs {
  message?: string;
  url?: URL;
}

/** Thrown when a network target is unreachable. */
export class Unreachable extends FreighterError.sub("unreachable") {
  url: URL;

  constructor(args: UnreachableArgs = {}) {
    const { message = "Unreachable", url = URL.UNKNOWN } = args;
    super(message);
    this.url = url;
  }
}

const freighterErrorEncoder: errors.Encoder = (error: errors.Typed) => {
  if (!error.type.startsWith(FreighterError.TYPE)) return null;
  if (EOF.matches(error)) return { type: EOF.TYPE, data: "EOF" };
  if (StreamClosed.matches(error))
    return { type: StreamClosed.TYPE, data: "StreamClosed" };
  if (Unreachable.matches(error))
    return { type: Unreachable.TYPE, data: "Unreachable" };
  throw new Error(`Unknown error type: ${error.type}: ${error.message}`);
};

const freighterErrorDecoder: errors.Decoder = (encoded: errors.Payload) => {
  if (!encoded.type.startsWith(FreighterError.TYPE)) return null;
  switch (encoded.type) {
    case EOF.TYPE:
      return new EOF();
    case StreamClosed.TYPE:
      return new StreamClosed();
    case Unreachable.TYPE:
      return new Unreachable();
    default:
      throw new errors.Unknown(`Unknown error type: ${encoded.data}`);
  }
};

errors.register({
  encode: freighterErrorEncoder,
  decode: freighterErrorDecoder,
});
