import { encode } from "msgpackr";
import { describe, expect, test } from "vitest";

import {
  BaseTypedError,
  EOF,
  FREIGHTER,
  NONE,
  StreamClosed,
  TypedError,
  UNKNOWN,
  UnknownError,
  Unreachable,
  assertErrorType,
  decodeError,
  encodeError,
  isTypedError,
  registerError,
} from "./errors";

class MyCustomError extends BaseTypedError {
  constructor(message: string) {
    super(message, "MyCustomError");
  }
}

const myCustomErrorEncoder = (error: MyCustomError): string => {
  return error.message;
};

const myCustomErrorDecoder = (encoded: string): TypedError => {
  return new MyCustomError(encoded);
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
      type: "MyCustomError",
      encode: myCustomErrorEncoder,
      decode: myCustomErrorDecoder,
    });
    const error = new MyCustomError("test");
    const encoded = encodeError(error);
    expect(encoded.type).toEqual("MyCustomError");
    expect(encoded.data).toEqual("test");
    const decoded = assertErrorType<MyCustomError>(
      "MyCustomError",
      decodeError(encoded)
    );
    expect(decoded.message).toEqual("test");
  });

  test("encoding and decoding a null error", () => {
    const encoded = encodeError(null);
    expect(encoded.type).toEqual(NONE);
    expect(encoded.data).toEqual("");
    const decoded = decodeError(encoded);
    expect(decoded).toBeUndefined();
  });

  test("encoding and decoding an unrecognized error", () => {
    const error = new Error("test");
    const encoded = encodeError(error);
    expect(encoded.type).toEqual(UNKNOWN);
    expect(encoded.data).toEqual("{}");
    const decoded = decodeError(encoded);
    expect(decoded).toEqual(new UnknownError("{}"));
  });

  test("registering duplicate error should throw", () => {
    registerError({
      type: "MyDuplicateError",
      encode: myCustomErrorEncoder,
      decode: myCustomErrorDecoder,
    });
    expect(() => {
      registerError({
        type: "MyDuplicateError",
        encode: myCustomErrorEncoder,
        decode: myCustomErrorDecoder,
      });
    }).toThrow();
  });

  test("encoding and decoding freighter errors", () => {
    [new EOF(), new StreamClosed(), new Unreachable()].forEach((error) => {
      const encoded = encodeError(error);
      expect(encoded.type).toEqual(FREIGHTER);
      expect(encoded.data).toEqual(error.message);
      const decoded = decodeError(encoded);
      expect(decoded).toEqual(error);
    });
  });
});
