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
});
