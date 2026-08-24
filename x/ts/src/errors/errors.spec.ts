// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import { errors } from "@/errors";

class ErrorOne extends errors.createTyped("one") {}

class ErrorTwo extends errors.createTyped("two") {}

class SubError extends ErrorOne.sub("child") {}

const myCustomErrorEncoder = (error: errors.Typed): errors.Payload | null => {
  if (error.type !== "one") return null;
  return { type: "one", data: error.message };
};

const myCustomErrorDecoder = (encoded: errors.Payload): errors.Typed =>
  new ErrorOne(encoded.data);

describe("errors", () => {
  describe("isTypedError", () => {
    it("should return true if the error implements the TypedError interface", () => {
      const error = new ErrorOne("test");
      const fError = errors.isTyped(error);
      expect(fError).toBe(true);
      expect(error.type).toEqual("one");
    });

    it("should return false if the error does not implement the TypedError interface", () => {
      const error = new Error("rando");
      const fError = errors.isTyped(error);
      expect(fError).toBe(false);
    });
  });

  describe("encoding/decoding", () => {
    it("should encode and decode a custom error through the registry", () => {
      errors.register({
        encode: myCustomErrorEncoder,
        decode: myCustomErrorDecoder,
      });
      const error = new ErrorOne("test");
      const encoded = errors.encode(error);
      const decoded = errors.decode(encoded);
      expect(ErrorOne.matches(decoded)).toBe(true);
    });

    test("should correctly encode/decode a null error", () => {
      const encoded = errors.encode(null);
      expect(encoded.type).toEqual(errors.NONE);
      expect(encoded.data).toEqual("");
      const decoded = errors.decode(encoded);
      expect(decoded).toBeNull();
    });

    it("should correctly encode/decode an undefined error", () => {
      const encoded = errors.encode(undefined);
      expect(encoded.type).toEqual(errors.NONE);
      expect(encoded.data).toEqual("");
      const decoded = errors.decode(encoded);
      expect(decoded).toBeNull();
    });

    it("should correctly encode/decode a generic error", () => {
      const error = new Error("test");
      const encoded = errors.encode(error);
      expect(encoded.type).toEqual(errors.UNKNOWN);
      expect(encoded.data).toEqual("test");
      const decoded = errors.decode(encoded);
      expect(decoded).toBeInstanceOf(Error);
      expect((decoded as Error).message).toEqual(error.message);
    });

    it("should correctly encode/decode a string that is not an error", () => {
      const error = "test";
      const encoded = errors.encode(error);
      expect(encoded.type).toEqual(errors.UNKNOWN);
      expect(encoded.data).toEqual("test");
      const decoded = errors.decode(encoded);
      expect(errors.Unknown.matches(decoded)).toBe(true);
    });

    it("should correctly encode/decode a random object", () => {
      const error = { foo: "bar" };
      const encoded = errors.encode(error);
      expect(encoded.type).toEqual(errors.UNKNOWN);
      expect(encoded.data).toEqual(JSON.stringify(error));
      const decoded = errors.decode(encoded);
      expect(errors.Unknown.matches(decoded)).toBe(true);
    });

    it("should preserve name and stack across encode/decode for a generic Error", () => {
      const error = new TypeError("boom");
      const encoded = errors.encode(error);
      expect(encoded.name).toEqual("TypeError");
      expect(encoded.stack).toEqual(error.stack);
      const decoded = errors.decode(encoded);
      expect((decoded as Error).name).toEqual("TypeError");
      expect((decoded as Error).stack).toEqual(error.stack);
    });

    it("should preserve name and stack when a typed error round-trips through a provider", () => {
      errors.register({
        encode: myCustomErrorEncoder,
        decode: myCustomErrorDecoder,
      });
      const error = new ErrorOne("boom");
      const encoded = errors.encode(error);
      expect(encoded.name).toEqual(ErrorOne.TYPE);
      expect(encoded.stack).toEqual(error.stack);
      const decoded = errors.decode(encoded);
      expect(ErrorOne.matches(decoded)).toBe(true);
      expect((decoded as Error).stack).toEqual(error.stack);
    });

    it("should leave the decoded error untouched when the payload omits name and stack", () => {
      const decoded = errors.decode({ type: errors.UNKNOWN, data: "no native fields" });
      expect(decoded).toBeInstanceOf(Error);
      expect((decoded as Error).message).toEqual("no native fields");
    });
  });

  describe("fromUnknown", () => {
    it("should return the same Error instance when given an Error", () => {
      const original = new Error("already an error");
      const result = errors.fromUnknown(original);
      expect(result).toBe(original);
    });

    it("should preserve typed-error subclasses when given one", () => {
      const original = new ErrorOne("typed");
      const result = errors.fromUnknown(original);
      expect(result).toBe(original);
      expect(ErrorOne.matches(result)).toBe(true);
    });

    it("should preserve other Error subclasses like TypeError", () => {
      const original = new TypeError("bad type");
      const result = errors.fromUnknown(original);
      expect(result).toBe(original);
      expect(result).toBeInstanceOf(TypeError);
    });

    it("should wrap a string with the string as the message", () => {
      const result = errors.fromUnknown("plain string");
      expect(result).toBeInstanceOf(Error);
      expect(result.message).toEqual('"plain string"');
      expect(result.cause).toEqual("plain string");
    });

    it("should wrap a number using JSON.stringify for the message", () => {
      const result = errors.fromUnknown(42);
      expect(result).toBeInstanceOf(Error);
      expect(result.message).toEqual("42");
      expect(result.cause).toEqual(42);
    });

    it("should wrap an object using JSON.stringify for the message", () => {
      const value = { foo: "bar", n: 1 };
      const result = errors.fromUnknown(value);
      expect(result).toBeInstanceOf(Error);
      expect(result.message).toEqual('{"foo":"bar","n":1}');
      expect(result.cause).toBe(value);
    });

    it("should fall back to String() when JSON.stringify throws on a circular ref", () => {
      const value: Record<string, unknown> = { name: "loop" };
      value.self = value;
      const result = errors.fromUnknown(value);
      expect(result).toBeInstanceOf(Error);
      // Plain objects without a custom toString fall through to "[object Object]".
      expect(result.message).toEqual("[object Object]");
      expect(result.cause).toBe(value);
    });

    it("should fall back to String() for a BigInt", () => {
      const value = 9007199254740993n;
      const result = errors.fromUnknown(value);
      expect(result).toBeInstanceOf(Error);
      expect(result.message).toEqual(String(value));
      expect(result.cause).toBe(value);
    });

    it("should wrap null with a stringified message", () => {
      const result = errors.fromUnknown(null);
      expect(result).toBeInstanceOf(Error);
      expect(result.message).toEqual("null");
      expect(result.cause).toBeNull();
    });

    it("should fall back to String() for undefined (JSON.stringify returns undefined)", () => {
      const result = errors.fromUnknown(undefined);
      expect(result).toBeInstanceOf(Error);
      expect(result.message).toEqual("undefined");
      expect(result.cause).toBeUndefined();
    });

    it("should fall back to String() for a function (JSON.stringify returns undefined)", () => {
      const value = function named() {};
      const result = errors.fromUnknown(value);
      expect(result).toBeInstanceOf(Error);
      expect(result.message).toEqual(String(value));
      expect(result.cause).toBe(value);
    });

    it("should fall back to String() for a symbol (JSON.stringify returns undefined)", () => {
      const value = Symbol("sym");
      const result = errors.fromUnknown(value);
      expect(result).toBeInstanceOf(Error);
      expect(result.message).toEqual("Symbol(sym)");
      expect(result.cause).toBe(value);
    });
  });

  describe("matches", () => {
    it("should return true if the errors are exactly the same", () => {
      const v = new ErrorOne("test");
      expect(ErrorOne.matches(v)).toBe(true);
      const v2 = new ErrorOne("test");
      expect(v2.matches(v)).toBe(true);
    });

    it("should return false if the errors are typed and clearly different", () => {
      const e1 = new ErrorOne("test");
      const e2 = new ErrorTwo("test");
      expect(e1.matches(e2)).toBe(false);
      expect(ErrorOne.matches(e2)).toBe(false);
      expect(ErrorTwo.matches(e1)).toBe(false);
    });

    it("should return false if the error is not typed", () => {
      const e1 = new ErrorOne("test");
      const e2 = new Error("rando");
      expect(e1.matches(e2)).toBe(false);
      expect(ErrorOne.matches(e2)).toBe(false);
    });

    it("should return false if the error is not actually an error", () => {
      const e1 = new ErrorOne("test");
      const e2 = "rando";
      expect(e1.matches(e2)).toBe(false);
      expect(ErrorOne.matches(e2)).toBe(false);
    });

    it("should return false if hte error is undefined", () => {
      const e1 = new ErrorOne("test");
      const e2 = undefined;
      expect(e1.matches(e2)).toBe(false);
      expect(ErrorOne.matches(e2)).toBe(false);
    });

    it("should return true if the the matching error is a parent", () => {
      const e1 = new ErrorOne("rando");
      const e2 = new SubError("rando");
      expect(e1.matches(e2)).toBe(true);
      expect(ErrorOne.matches(e1)).toBe(true);
    });

    it("should return false if the matching error is a sub-error", () => {
      const e1 = new ErrorOne("random");
      const e2 = new SubError("random");
      expect(e2.matches(e1)).toBe(false);
      expect(SubError.matches(e1)).toBe(false);
    });
  });

  describe("matchExact", () => {
    it("should return true when the type matches exactly", () => {
      const e = new ErrorOne("test");
      expect(ErrorOne.matchExact(e)).toBe(true);
    });

    it("should return false for a sub-error, unlike matches", () => {
      const sub = new SubError("test");
      expect(ErrorOne.matches(sub)).toBe(true);
      expect(ErrorOne.matchExact(sub)).toBe(false);
      expect(SubError.matchExact(sub)).toBe(true);
    });

    it("should return false for a different typed error", () => {
      expect(ErrorOne.matchExact(new ErrorTwo("test"))).toBe(false);
    });

    it("should return false for an untyped error, a raw string, or undefined", () => {
      expect(ErrorOne.matchExact(new Error("one"))).toBe(false);
      expect(ErrorOne.matchExact("one")).toBe(false);
      expect(ErrorOne.matchExact(undefined)).toBe(false);
    });

    it("should narrow an unknown value to the matched error", () => {
      const e: unknown = new ErrorOne("hello");
      if (!ErrorOne.matchExact(e)) throw new Error("expected an exact match");
      expect(e.type).toEqual("one");
      expect(e.message).toEqual("hello");
    });
  });
});
