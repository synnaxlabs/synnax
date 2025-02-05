// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, URL } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";
import { z } from "zod";

import {
  BaseTypedError,
  EOF,
  type ErrorPayload,
  registerError,
  type TypedError,
} from "@/errors";
import { type Context } from "@/middleware";
import { WebSocketClient } from "@/websocket";

const url = new URL({
  host: "127.0.0.1",
  port: 8080,
});

const MessageSchema = z.object({
  id: z.number().optional(),
  message: z.string().optional(),
});

const client = new WebSocketClient(url, new binary.JSONCodec());

class MyCustomError extends BaseTypedError {
  code: number;
  type = "integration.error";

  constructor(message: string, code: number) {
    super(message);
    this.code = code;
  }
}

const encodeTestError = (err: TypedError): ErrorPayload => {
  if (!(err instanceof MyCustomError)) throw new Error("Unexpected error type");

  return { type: "integration.error", data: `${err.code},${err.message}` };
};

const decodeTestError = (encoded: ErrorPayload): TypedError | null => {
  if (encoded.type !== "integration.error") return null;
  const [code, message] = encoded.data.split(",");
  return new MyCustomError(message, parseInt(code));
};

registerError({
  encode: encodeTestError,
  decode: decodeTestError,
});

describe("websocket", () => {
  test("basic exchange", async () => {
    const stream = await client.stream("stream/echo", MessageSchema, MessageSchema);
    for (let i = 0; i < 10; i++) {
      stream.send({ id: i, message: "hello" });
      const [response, error] = await stream.receive();
      expect(error).toBeNull();
      expect(response?.id).toEqual(i + 1);
      expect(response?.message).toEqual("hello");
    }
    stream.closeSend();
    const [response, error] = await stream.receive();
    expect(error).toEqual(new EOF());
    expect(response).toBeNull();
  });

  test("receive message after close", async () => {
    const stream = await client.stream(
      "stream/sendMessageAfterClientClose",
      MessageSchema,
      MessageSchema,
    );
    stream.closeSend();
    const [response, error] = await stream.receive();
    expect(error).toBeNull();
    expect(response?.id).toEqual(0);
    expect(response?.message).toEqual("Close Acknowledged");
    const [, recvError] = await stream.receive();
    expect(recvError).toEqual(new EOF());
  });

  test("receive error", async () => {
    const stream = await client.stream(
      "stream/receiveAndExitWithErr",
      MessageSchema,
      MessageSchema,
    );
    stream.send({ id: 0, message: "hello" });
    const [response, error] = await stream.receive();
    expect(error).toEqual(new MyCustomError("unexpected error", 1));
    expect(response).toBeNull();
  });

  describe("middleware", () => {
    test("receive middleware", async () => {
      const myClient = new WebSocketClient(url, new binary.JSONCodec());
      let c = 0;
      myClient.use(async (md, next): Promise<[Context, Error | null]> => {
        if (md.params !== undefined) {
          c++;
          md.params.Test = "test";
        }
        return await next(md);
      });
      await myClient.stream("stream/middlewareCheck", MessageSchema, MessageSchema);
      expect(c).toEqual(1);
    });

    test("middleware error on server", async () => {
      const myClient = new WebSocketClient(url, new binary.JSONCodec());
      await expect(
        myClient.stream("stream/middlewareCheck", MessageSchema, MessageSchema),
      ).rejects.toThrow("test param not found");
    });
  });
});
