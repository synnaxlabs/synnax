// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type IncomingMessage } from "node:http";
import { type AddressInfo } from "node:net";

import { binary, errors, sleep, TimeSpan, url } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, test } from "vitest";
import { type WebSocket, WebSocketServer } from "ws";
import { z } from "zod";

import {
  type Context,
  EOF,
  type FlowControl,
  ReceiveOverflow,
  WebSocketClient,
} from "@/index";

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
      await stream.send({ id: i, message: "hello" });
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
    await stream.send({ id: 0, message: "hello" });
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

describe("flow control", () => {
  interface Connection {
    ws: WebSocket;
    req: IncomingMessage;
  }

  let server: WebSocketServer;
  let connected: Promise<Connection>;

  const dial = async (flowControl: Partial<FlowControl>) => {
    const { port } = server.address() as AddressInfo;
    const c = new WebSocketClient(
      new url.URL({ host: "127.0.0.1", port }),
      new binary.JSONCodec(),
      false,
      flowControl,
    );
    return await c.stream("stream", messageSchema, messageSchema);
  };

  const sendData = (ws: WebSocket, payload: z.infer<typeof messageSchema>) =>
    ws.send(Buffer.from(JSON.stringify({ type: "data", payload })));

  beforeEach(async () => {
    server = new WebSocketServer({ port: 0, host: "127.0.0.1" });
    connected = new Promise((resolve) =>
      server.on("connection", (ws, req) => {
        ws.send(Buffer.from(JSON.stringify({ type: "open", payload: null })));
        resolve({ ws, req });
      }),
    );
    await new Promise((resolve) => server.on("listening", resolve));
  });

  afterEach(async () => {
    server.clients.forEach((ws) => ws.terminate());
    await new Promise((resolve) => server.close(resolve));
  });

  test("should receive queued messages in order", async () => {
    const stream = await dial({});
    const { ws } = await connected;
    for (let id = 0; id < 10; id++) sendData(ws, { id });
    for (let id = 0; id < 10; id++) expect((await stream.receive()).id).toEqual(id);
  });

  test("should fail the stream when unread messages pass the limit", async () => {
    const stream = await dial({ receiveQueueLimit: 1024 });
    const { ws } = await connected;
    const closed = new Promise((resolve) => ws.on("close", resolve));
    const message = "x".repeat(256);
    for (let id = 0; id < 10; id++) sendData(ws, { id, message });
    await closed;
    await expect(stream.send({ id: 0 })).rejects.toThrow(EOF);
    await expect(stream.receive()).rejects.toThrow(ReceiveOverflow);
  });

  test("should wait to send while the socket buffer is full", async () => {
    const stream = await dial({ sendHighWaterMark: 1024 });
    const { ws, req } = await connected;
    req.socket.pause();
    const received: number[] = [];
    const codec = new binary.JSONCodec();
    const msgZ = z.object({ payload: messageSchema });
    ws.on("message", (data: Buffer) => {
      received.push(codec.decode(data, msgZ).payload.id ?? -1);
    });
    await stream.send({ id: 0, message: "x".repeat(32 * 1024 * 1024) });
    let sent = false;
    const second = stream.send({ id: 1 }).then(() => (sent = true));
    await sleep.sleep(TimeSpan.milliseconds(50));
    expect(sent).toBe(false);
    req.socket.resume();
    await second;
    await expect.poll(() => received).toEqual([0, 1]);
  });
});
