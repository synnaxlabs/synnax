// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import {
  assertErrorType,
  BaseTypedError,
  decodeError,
  encodeError,
  EOF,
  type ErrorPayload,
  FREIGHTER,
  isTypedError,
  NONE,
  registerError,
  StreamClosed,
  type TypedError,
  UNKNOWN,
  UnknownError,
  Unreachable,
} from "@/errors";

class MyCustomError extends BaseTypedError {
  type = "MyCustomError";
  constructor(message: string) {
    super(message);
  }
}

const myCustomErrorEncoder = (error: MyCustomError): ErrorPayload => {
  return { type: "MyCustomError", data: error.message };
};

const myCustomErrorDecoder = (encoded: ErrorPayload): TypedError => {
  return new MyCustomError(encoded.data);
};

describe("errors", () => {
  test("isTypedError", () => {
    const error = new MyCustomError("test");
    const fError = isTypedError(error);
    expect(fError).toBeTruthy();
    expect(error.type).toEqual("MyCustomError");
  });

  test("encoding and decoding a custom error through registry", () => {
    registerError({
      encode: myCustomErrorEncoder,
      decode: myCustomErrorDecoder,
    });
    const error = new MyCustomError("test");
    const encoded = encodeError(error);
    expect(encoded.type).toEqual("MyCustomError");
    expect(encoded.data).toEqual("test");
    const decoded = assertErrorType<MyCustomError>(
      "MyCustomError",
      decodeError(encoded),
    );
    expect(decoded.message).toEqual("test");
  });

  test("encoding and decoding a null error", () => {
    const encoded = encodeError(null);
    expect(encoded.type).toEqual(NONE);
    expect(encoded.data).toEqual("");
    const decoded = decodeError(encoded);
    expect(decoded).toBeNull();
  });

  test("encoding and decoding an unrecognized error", () => {
    const error = new Error("test");
    const encoded = encodeError(error);
    expect(encoded.type).toEqual(UNKNOWN);
    expect(encoded.data).toEqual("{}");
    const decoded = decodeError(encoded);
    expect(decoded).toEqual(new UnknownError("{}"));
  });

  test("encoding and decoding freighter errors", () => {
    [new EOF(), new StreamClosed(), new Unreachable()].forEach((error) => {
      const encoded = encodeError(error);
      expect(encoded.type.startsWith(FREIGHTER)).toBeTruthy();
      expect(encoded.data).toEqual(error.message);
      const decoded = decodeError(encoded);
      expect(decoded).toEqual(error);
    });
  });
});
