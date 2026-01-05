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

/** an error type that can match against other errors. */
export interface Matchable {
  /**
   * @returns true if the provided error matches the matchable.
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
   * @param message - the error message.
   * @param options - the error options.
   * @returns a new TypedError.
   */
  new (message?: string, options?: ErrorOptions): Typed;
  /**
   * the type of the error.
   */
  TYPE: string;
  /**
   * creates a new subclass of the error that extends its type. So if the type of this
   * class is `dog` and subType is `labrador`, the type of the new class will be
   * `dog.labrador`.
   * @param subType - the type of the new error.
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

    constructor(message?: string, options?: ErrorOptions) {
      super(message, options);
      this.name = Internal.TYPE;
    }
    static sub(subType: string): TypedClass {
      return createTyped(`${type}.${subType}`);
    }
  };

/**
 * Function that decodes an encoded error payload back into an error object
 * @param encoded - The encoded error payload to decode
 * @returns The decoded error object or null if the decoder cannot handle this error
 * type
 */
export type Decoder = (encoded: Payload) => Error | null;

/**
 * Function that encodes a typed error into a network-portable payload
 * @param error - The typed error to encode
 * @returns The encoded error payload or null if the encoder cannot handle this error type
 */
export type Encoder = (error: Typed) => Payload | null;

/**
 * Checks if an unknown value is a TypedError
 * @param error - The value to check
 * @returns True if the value is a TypedError, false otherwise
 */
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

/** Constant representing an unknown error type */
export const UNKNOWN = "unknown";

/** Constant representing no error (null) */
export const NONE = "nil";

/**
 * provides custom encoding/decoding mechanisms for specific error
 * categories.
 */
interface Provider {
  /**
   * Encodes an error into a primitive payload that can be sent over the network or stored
   * on disk.
   * @param error - The error to encode.
   * @returns The encoded error.
   */
  encode: Encoder;
  /**
   * Decodes an error from a primitive payload that can be sent over the network or stored
   * on disk.
   * @param payload - The encoded error.
   * @returns The decoded error.
   */
  decode: Decoder;
}

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
        if (payload != null) return payload;
      }
    if (error instanceof Error) return { type: UNKNOWN, data: error.message };
    if (typeof error === "string") return { type: UNKNOWN, data: error };
    try {
      return { type: UNKNOWN, data: JSON.stringify(error) };
    } catch {
      return { type: UNKNOWN, data: "unable to encode error information" };
    }
  }

  decode(payload?: Payload | null): Error | null {
    if (payload == null || payload.type === NONE) return null;
    if (payload.type === UNKNOWN) return new Unknown(payload.data);
    for (const provider of this.providers) {
      const error = provider.decode(payload);
      if (error != null) return error;
    }
    return new Unknown(payload.data);
  }
}

const getRegistry = singleton.define("synnax-error-registry", () => new Registry());

/**
 * Registers a custom error type with the error registry, which allows it to be
 * encoded/decoded and sent over the network.
 *
 * @param type - A unique string identifier for the error type.
 * @param encode - A function that encodes the error into a string.
 * @param decode - A function that decodes the error from a string.
 */
export const register = ({ encode, decode }: Provider): void =>
  getRegistry().register({ encode, decode });

/**
 * Encodes an error into a primitive payload that can be sent over the network or stored
 * on disk.
 * @param error - The error to encode.
 * @returns The encoded error.
 */
export const encode = (error: unknown): Payload => getRegistry().encode(error);

/**
 * Decodes an error payload into an exception. If a custom decoder can be found
 * for the error type, it will be used. Otherwise, a generic Error containing
 * the error data is returned.
 *
 * @param payload - The encoded error payload.
 * @returns The decoded error.
 */
export const decode = (payload?: Payload | null): Error | null => {
  if (payload == null) return null;
  return getRegistry().decode(payload);
};

/**
 * Generic error for representing unknown errors
 */
export class Unknown extends createTyped("unknown") {}

/** Zod schema for validating error payloads */
export const payloadZ = z.object({ type: z.string(), data: z.string() });

/** Network-portable representation of an error */
export type Payload = z.infer<typeof payloadZ>;

/** Error for representing the cancellation of an operation */
export class Canceled extends createTyped("canceled") {}

/** A payload representing a native JavaScript Error */
export interface NativePayload {
  /** The name of the error */
  name: string;
  /** The message of the error */
  message: string;
  /** The stack trace of the error */
  stack?: string;
}

/** Error for representing a method that is not implemented */
export class NotImplemented extends createTyped("not_implemented") {}
