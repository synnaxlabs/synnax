// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, errors, url } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";
import { z } from "zod";

import { EOF } from "@/errors";
import { type Context } from "@/middleware";
import { WebSocketClient } from "@/websocket";

const endpoint = new url.URL({ host: "127.0.0.1", port: 8080 });

const messageSchema = z.object({
  id: z.number().optional(),
  message: z.string().optional(),
});

const client = new WebSocketClient(endpoint, new binary.JSONCodec());

class MyCustomError extends errors.createTyped("integration.error") {
  code: number;
  constructor(message: string, code: number) {
    super(message);
    this.code = code;
  }
}

const encodeTestError = (err: errors.Typed): errors.Payload => {
  if (!(err instanceof MyCustomError)) throw new Error("Unexpected error type");

  return { type: "integration.error", data: `${err.code},${err.message}` };
};

const decodeTestError = (encoded: errors.Payload): errors.Typed | null => {
  if (encoded.type !== "integration.error") return null;
  const [code, message] = encoded.data.split(",");
  return new MyCustomError(message, parseInt(code, 10));
};

errors.register({ encode: encodeTestError, decode: decodeTestError });

describe("websocket", () => {
  test("basic exchange", async () => {
    const stream = await client.stream("stream/echo", messageSchema, messageSchema);
    for (let i = 0; i < 10; i++) {
      stream.send({ id: i, message: "hello" });
      const response = await stream.receive();
      expect(response.id).toEqual(i + 1);
      expect(response.message).toEqual("hello");
    }
    stream.closeSend();
    await expect(stream.receive()).rejects.toThrow(EOF);
  });

  test("receive message after close", async () => {
    const stream = await client.stream(
      "stream/sendMessageAfterClientClose",
      messageSchema,
      messageSchema,
    );
    stream.closeSend();
    const response = await stream.receive();
    expect(response.id).toEqual(0);
    expect(response.message).toEqual("Close Acknowledged");
    await expect(stream.receive()).rejects.toThrow(EOF);
  });

  test("receive error", async () => {
    const stream = await client.stream(
      "stream/receiveAndExitWithErr",
      messageSchema,
      messageSchema,
    );
    stream.send({ id: 0, message: "hello" });
    await expect(stream.receive()).rejects.toThrow(MyCustomError);
  });

  describe("middleware", () => {
    test("receive middleware", async () => {
      const myClient = new WebSocketClient(endpoint, new binary.JSONCodec());
      let c = 0;
      myClient.use(async (md, next): Promise<Context> => {
        if (md.params !== undefined) {
          c++;
          md.params.Test = "test";
        }
        return await next(md);
      });
      await myClient.stream("stream/middlewareCheck", messageSchema, messageSchema);
      expect(c).toEqual(1);
    });

    test("middleware error on server", async () => {
      const myClient = new WebSocketClient(endpoint, new binary.JSONCodec());
      await expect(
        myClient.stream("stream/middlewareCheck", messageSchema, messageSchema),
      ).rejects.toThrow("test param not found");
    });
  });
});
