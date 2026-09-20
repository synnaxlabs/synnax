// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { singleton } from "@/singleton";

const ERROR_DISCRIMINATOR = "sy_x_error";

/**
 * @returns general function that returns true if an error matches a set of
 * abstracted criteria
 */
export type Matcher = (e: unknown) => boolean;

/**
 * A type guard that narrows a value to a {@link Typed} error when its type is exactly
 * the matcher's type. Unlike {@link Matcher}, an exact match guarantees identity, so it
 * can soundly narrow. Errors that carry extra structured fields may override their
 * `matchExact` to narrow to their concrete type.
 */
export type ExactMatcher = (e: unknown) => e is Typed;

/** an error type that can match against other errors. */
export interface Matchable {
  /**
   * @returns true if the provided error matches the matchable by type prefix, message,
   * or raw string. This is a fuzzy membership test for control flow, not a type guard.
   */
  matches: Matcher;
}

/**
 * An error that has a network-portable type, allowing it to be encoded/decoded into
 * a JSON representation. Also allows for simpler matching using @method matches instead of using
 * instanceof, which has a number of caveats.
 */
export interface Typed extends Error, Matchable {
  discriminator: typeof ERROR_DISCRIMINATOR;
  /**
   * Returns a unique type identifier for the error. The errors package uses this to
   *  determine the correct decoder to use when encoding/decoding errors.
   */
  type: string;
}

/**
 * a class that, when constructed, implements the TypedError interface. Also provides
 * utilities for matching and creating subclasses.
 */
export interface TypedClass extends Matchable {
  /**
   * constructs a new TypedError. Identical to the Error constructor.
   * @returns a new TypedError.
   */
  new (message?: string, options?: ErrorOptions): Typed;
  /**
   * Narrows a value to this error type when its type is exactly this class's type.
   * Errors carrying extra structured fields may override this to narrow to their
   * concrete type.
   */
  matchExact: ExactMatcher;
  /** the type of the error. */
  TYPE: string;
  /**
   * creates a new subclass of the error that extends its type. So if the type of this
   * class is `dog` and subType is `labrador`, the type of the new class will be
   * `dog.labrador`.
   * @returns a new TypedErrorClass.
   */
  sub: (subType: string) => TypedClass;
}

/**
 * @param type - the error type to match
 * @returns a function that matches errors of the given type. Returns true if
 * the provided instance of Error or a string message contains the provided error type.
 */
const createTypeMatcher =
  (type: string): Matcher =>
  (e) => {
    if (e != null && typeof e === "object" && "type" in e && typeof e.type === "string")
      return e.type.startsWith(type);
    if (e instanceof Error) return e.message.startsWith(type);
    if (typeof e !== "string") return false;
    return e.startsWith(type);
  };

/**
 * @param type - the error type to match exactly.
 * @returns a type guard that narrows a value to a {@link Typed} error when its type is
 * exactly the given type.
 */
const createExactMatcher =
  (type: string): ExactMatcher =>
  (e): e is Typed =>
    e != null &&
    typeof e === "object" &&
    "type" in e &&
    typeof e.type === "string" &&
    e.type === type;

/**
 * Creates a new class definition that implements the TypedErrorClass interface.
 * @param type - the type of the error.
 * @returns a new TypedErrorClass.
 * @example
 * ```ts
 * class MyError extends createTypedError("my_error") {}
 * ```
 */
export const createTyped = (type: string): TypedClass =>
  class Internal extends Error implements Typed {
    static readonly discriminator = ERROR_DISCRIMINATOR;
    readonly discriminator = Internal.discriminator;

    static readonly TYPE = type;
    readonly type: string = Internal.TYPE;

    static readonly matches = createTypeMatcher(type);
    readonly matches: Matcher = Internal.matches;

    static readonly matchExact = createExactMatcher(type);

    constructor(message?: string, options?: ErrorOptions) {
      super(message, options);
      this.name = Internal.TYPE;
    }
    static sub(subType: string): TypedClass {
      return createTyped(`${type}.${subType}`);
    }
  };

/** Decodes a payload, or returns null when it cannot handle the payload's type. */
export type Decoder = (encoded: Payload) => Error | null;

/** Encodes a typed error, or returns null when it cannot handle the error's type. */
export type Encoder = (error: Typed) => Payload | null;

/** @returns true if the value is a {@link Typed} error. */
export const isTyped = (error: unknown): error is Typed => {
  if (error == null || typeof error !== "object") return false;
  const typedError = error as Typed;
  if (typedError.discriminator !== ERROR_DISCRIMINATOR) return false;
  if (!("type" in typedError))
    throw new Error(
      `X Error is missing its type property: ${JSON.stringify(typedError)}`,
    );
  return true;
};

/**
 * Coerces an arbitrary thrown value into an `Error` so it can be re-thrown without
 * tripping `@typescript-eslint/only-throw-error` and so callers can rely on a uniform
 * `Error` shape. The original value is preserved on `Error.cause` for stack-trace
 * continuity.
 *
 * - If `value` is already an `Error`, it is returned unchanged.
 * - Otherwise the message is derived from `JSON.stringify(value)` when possible (which
 *   carries more detail for plain objects), falling back to `String(value)` for
 *   circular structures, BigInts, or anything else that fails to serialize.
 */
export const fromUnknown = (value: unknown): Error => {
  if (value instanceof Error) return value;
  let message: string;
  try {
    message = JSON.stringify(value) ?? String(value);
  } catch {
    message = String(value);
  }
  return new Error(message, { cause: value });
};

/** Constant representing an unknown error type */
export const UNKNOWN = "unknown";

/** Constant representing no error (null) */
export const NONE = "nil";

/** provides custom encoding/decoding mechanisms for specific error categories. */
interface Provider {
  /** Encodes an error into a payload for the network or disk. */
  encode: Encoder;
  /** Decodes an error from a payload read off the network or disk. */
  decode: Decoder;
}

const withNative = (payload: Payload, error: Error): Payload => ({
  ...payload,
  name: error.name,
  stack: error.stack,
});

const applyNative = (error: Error, payload: Payload): Error => {
  if (payload.name != null) error.name = payload.name;
  if (payload.stack != null) error.stack = payload.stack;
  return error;
};

class Registry {
  private readonly providers: Provider[] = [];

  register(provider: Provider): void {
    this.providers.push(provider);
  }

  encode(error: unknown): Payload {
    if (error == null) return { type: NONE, data: "" };
    if (isTyped(error))
      for (const provider of this.providers) {
        const payload = provider.encode(error);
        if (payload != null) return withNative(payload, error);
      }
    if (error instanceof Error)
      return withNative({ type: UNKNOWN, data: error.message }, error);
    if (typeof error === "string") return { type: UNKNOWN, data: error };
    try {
      return { type: UNKNOWN, data: JSON.stringify(error) };
    } catch {
      return { type: UNKNOWN, data: "unable to encode error information" };
    }
  }

  decode(payload?: Payload | null): Error | null {
    if (payload == null || payload.type === NONE) return null;
    if (payload.type === UNKNOWN)
      return applyNative(new Unknown(payload.data), payload);
    for (const provider of this.providers) {
      const error = provider.decode(payload);
      if (error != null) return applyNative(error, payload);
    }
    return applyNative(new Unknown(payload.data), payload);
  }
}

const getRegistry = singleton.define("synnax-error-registry", () => new Registry());

/** Registers an encode/decode pair so its error type survives the wire. */
export const register = ({ encode, decode }: Provider): void =>
  getRegistry().register({ encode, decode });

/** Encodes an error into a payload for the network or disk. */
export const encode = (error: unknown): Payload => getRegistry().encode(error);

/**
 * Decodes a payload into an error, through a registered decoder for its type when one
 * exists and a generic Error carrying the payload's data otherwise.
 */
export const decode = (payload?: Payload | null): Error | null => {
  if (payload == null) return null;
  return getRegistry().decode(payload);
};

/** Generic error for representing unknown errors */
export class Unknown extends createTyped("unknown") {}

/** Zod schema for validating error payloads. `name` and `stack` are TypeScript-only
 * fields; Go and Python don't populate them. They're carried opaquely across the wire
 * and re-applied to the reconstructed error on decode, which keeps the original error
 * name (e.g. "TypeError") and stack trace alive across worker / network boundaries. */
export const payloadZ = z.object({
  type: z.string(),
  data: z.string(),
  name: z.string().optional(),
  stack: z.string().optional(),
});

/** Network-portable representation of an error */
export type Payload = z.infer<typeof payloadZ>;

/** Error for representing the cancellation of an operation */
export class Canceled extends createTyped("canceled") {}

/** Error for representing a method that is not implemented */
export class NotImplemented extends createTyped("not_implemented") {}
