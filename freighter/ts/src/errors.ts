// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { URL } from "@synnaxlabs/x";
import { z } from "zod";

/** Basic interface for an error or error type that can be matched against a candidate error */
export interface MatchableErrorType {
  /**
   * @returns a function that matches errors of the given type. Returns true if
   * the provided instance of Error or a string message contains the provided error type.
   */
  matches: (e: string | Error | unknown) => boolean;
}

export interface TypedError extends Error {
  discriminator: "FreighterError";
  /**
   * @description Returns a unique type identifier for the error. Freighter uses this to
   * determine the correct decoder to use on the other end of the freighter.
   */
  type: string;
}

/**
 * @param type - the error type to match
 * @returns a function that matches errors of the given type. Returns true if
 * the provided instance of Error or a string message contains the provided error type.
 */
export const errorMatcher =
  (type: string) =>
  (e: string | Error | unknown): boolean => {
    if (e != null && typeof e === "object" && "type" in e && typeof e.type === "string")
      return e.type.includes(type);
    if (e instanceof Error) return e.message.includes(type);
    if (typeof e !== "string") return false;
    return e.includes(type);
  };

export class BaseTypedError extends Error implements TypedError {
  readonly discriminator = "FreighterError";
  type: string = "";
}

type ErrorDecoder = (encoded: ErrorPayload) => Error | null;
type ErrorEncoder = (error: TypedError) => ErrorPayload | null;

export const isTypedError = (error: unknown): error is TypedError => {
  if (error == null || typeof error !== "object") return false;
  const typedError = error as TypedError;
  if (typedError.discriminator !== "FreighterError") return false;
  if (!("type" in typedError))
    throw new Error(
      `Freighter error is missing its type property: ${JSON.stringify(typedError)}`,
    );
  return true;
};

export const assertErrorType = <T>(type: string, error?: Error | null): T => {
  if (error == null)
    throw new Error(`Expected error of type ${type} but got nothing instead`);
  if (!isTypedError(error))
    throw new Error(`Expected a typed error, got: ${error.message}`);
  if (error.type !== type)
    throw new Error(
      `Expected error of type ${type}, got ${error.type}: ${error.message}`,
    );
  return error as unknown as T;
};

export const UNKNOWN = "unknown";
export const NONE = "nil";
export const FREIGHTER = "freighter";

export const errorZ = z.object({ type: z.string(), data: z.string() });

export type ErrorPayload = z.infer<typeof errorZ>;

interface errorProvider {
  encode: ErrorEncoder;
  decode: ErrorDecoder;
}

class Registry {
  private readonly providers: errorProvider[] = [];

  register(provider: errorProvider): void {
    this.providers.push(provider);
  }

  encode(error: unknown): ErrorPayload {
    if (error == null) return { type: NONE, data: "" };
    if (isTypedError(error))
      for (const provider of this.providers) {
        const payload = provider.encode(error);
        if (payload != null) return payload;
      }

    return { type: UNKNOWN, data: JSON.stringify(error) };
  }

  decode(payload?: ErrorPayload | null): Error | null {
    if (payload == null || payload.type === NONE) return null;
    if (payload.type === UNKNOWN) return new UnknownError(payload.data);
    for (const provider of this.providers) {
      const error = provider.decode(payload);
      if (error != null) return error;
    }
    return new UnknownError(payload.data);
  }
}

const REGISTRY = new Registry();

/**
 * Registers a custom error type with the error registry, which allows it to be
 * encoded/decoded and sent over the network.
 *
 * @param type - A unique string identifier for the error type.
 * @param encode - A function that encodes the error into a string.
 * @param decode - A function that decodes the error from a string.
 */
export const registerError = ({
  encode,
  decode,
}: {
  encode: ErrorEncoder;
  decode: ErrorDecoder;
}): void => REGISTRY.register({ encode, decode });

/**
 * Encodes an error into a payload that can be sent between a freighter server
 * and client.
 * @param error - The error to encode.
 * @returns The encoded error.
 */
export const encodeError = (error: unknown): ErrorPayload => REGISTRY.encode(error);

/**
 * Decodes an error payload into an exception. If a custom decoder can be found
 * for the error type, it will be used. Otherwise, a generic Error containing
 * the error data is returned.
 *
 * @param payload - The encoded error payload.
 * @returns The decoded error.
 */
export const decodeError = (payload: ErrorPayload): Error | null =>
  REGISTRY.decode(payload);

export class UnknownError extends BaseTypedError implements TypedError {
  type = "unknown";
}

const FREIGHTER_ERROR_TYPE = "freighter.";

/** Thrown/returned when a stream closed normally. */
export class EOF extends BaseTypedError implements TypedError {
  static readonly TYPE = `${FREIGHTER_ERROR_TYPE}eof`;
  type = EOF.TYPE;
  static readonly matches = errorMatcher(EOF.TYPE);

  constructor() {
    super("EOF");
  }
}

/** Thrown/returned when a stream is closed abnormally. */
export class StreamClosed extends BaseTypedError implements TypedError {
  static readonly TYPE = `${FREIGHTER_ERROR_TYPE}stream_closed`;
  static readonly matches = errorMatcher(StreamClosed.TYPE);
  type = StreamClosed.TYPE;

  constructor() {
    super("StreamClosed");
  }
}

export interface UnreachableArgs {
  message?: string;
  url?: URL;
}

/** Thrown when a target is unreachable. */
export class Unreachable extends BaseTypedError implements TypedError {
  static readonly TYPE = `${FREIGHTER_ERROR_TYPE}unreachable`;
  type = Unreachable.TYPE;
  static readonly matches = errorMatcher(Unreachable.TYPE);
  url: URL;

  constructor(args: UnreachableArgs = {}) {
    const { message = "Unreachable", url = URL.UNKNOWN } = args;
    super(message);
    this.url = url;
  }
}

const freighterErrorEncoder: ErrorEncoder = (error: TypedError) => {
  if (!error.type.startsWith(FREIGHTER)) return null;
  if (EOF.matches(error)) return { type: EOF.TYPE, data: "EOF" };
  if (StreamClosed.matches(error))
    return { type: StreamClosed.TYPE, data: "StreamClosed" };
  if (Unreachable.matches(error))
    return { type: Unreachable.TYPE, data: "Unreachable" };
  throw new Error(`Unknown error type: ${error.type}: ${error.message}`);
};

const freighterErrorDecoder: ErrorDecoder = (encoded: ErrorPayload) => {
  if (!encoded.type.startsWith(FREIGHTER_ERROR_TYPE)) return null;
  switch (encoded.type) {
    case EOF.TYPE:
      return new EOF();
    case StreamClosed.TYPE:
      return new StreamClosed();
    case Unreachable.TYPE:
      return new Unreachable();
    default:
      throw new Error(`Unknown error type: ${encoded.data}`);
  }
};

registerError({
  encode: freighterErrorEncoder,
  decode: freighterErrorDecoder,
});
