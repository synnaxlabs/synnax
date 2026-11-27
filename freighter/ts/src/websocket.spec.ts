// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { IncomingMessage } from "node:http";
import { type AddressInfo, createServer, type Socket as NetSocket } from "node:net";

import { binary, errors, sleep, TimeSpan, url } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { type WebSocket, WebSocketServer } from "ws";
import { z } from "zod";

import {
  type Connector,
  type Context,
  EOF,
  type FlowControl,
  type KeepAlive,
  ReceiveOverflow,
  type Socket,
  subprotocols,
  Timeout,
  Unreachable,
  UntrustedCertificate,
  WebSocketClient,
} from "@/index";

const endpoint = new url.URL({ host: "127.0.0.1", port: 8080 });

const messageSchema = z.object({
  id: z.number().optional(),
  message: z.string().optional(),
});

const client = new WebSocketClient({ endpoint, codec: new binary.JSONCodec() });

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
      const myClient = new WebSocketClient({ endpoint, codec: new binary.JSONCodec() });
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
      const myClient = new WebSocketClient({ endpoint, codec: new binary.JSONCodec() });
      await expect(
        myClient.stream("stream/middlewareCheck", messageSchema, messageSchema),
      ).rejects.toThrow("test param not found");
    });
  });
});

describe("unreachable server", () => {
  test("should throw Unreachable when nothing listens on the port", async () => {
    const server = createServer();
    await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
    const { port } = server.address() as AddressInfo;
    await new Promise((resolve) => server.close(resolve));
    const endpoint = new url.URL({ host: "127.0.0.1", port });
    const c = new WebSocketClient({ endpoint, codec: binary.JSON_CODEC });
    const err = await c.stream("stream", messageSchema, messageSchema).then(
      () => null,
      (e: unknown) => e,
    );
    expect(Unreachable.matches(err)).toBe(true);
    expect(err).toMatchObject({ url: { protocol: "ws", port, path: "stream/" } });
  });

  const failingConnector = (
    fail: (socket: Socket) => void,
    readyState: number = globalThis.WebSocket.CONNECTING,
  ): Connector => ({
    fetch: async () => await Promise.reject(new Error("not used")),
    dial: async () => {
      const socket: Socket = {
        binaryType: "",
        bufferedAmount: 0,
        readyState,
        send: () => {},
        close: () => {},
        onopen: null,
        onmessage: null,
        onclose: null,
        onerror: null,
      };
      setTimeout(() => fail(socket));
      return socket;
    },
  });

  const streamError = async (
    connector: Connector,
    secure = false,
  ): Promise<unknown> => {
    const c = new WebSocketClient({
      endpoint,
      codec: binary.JSON_CODEC,
      connector,
      secure,
    });
    return await c.stream("stream", messageSchema, messageSchema).then(
      () => null,
      (e: unknown) => e,
    );
  };

  const withCode = (message: string, code: string): Error =>
    Object.assign(new Error(message), { code });

  const networkError = (message: string): Error =>
    new DOMException(message, "NetworkError");

  test.each([
    [
      "a Bun refused dial",
      withCode("Failed to connect", "ECONNREFUSED"),
      "unreachable",
    ],
    [
      "a Bun untrusted certificate",
      withCode("TLS handshake failed", "UNABLE_TO_VERIFY_LEAF_SIGNATURE"),
      "untrusted",
    ],
    [
      "a Bun expired certificate",
      withCode("TLS handshake failed", "CERT_HAS_EXPIRED"),
      "runtime",
    ],
    ["a Deno refused dial", networkError("tcp connect error"), "unreachable"],
    [
      "a Deno untrusted certificate",
      networkError("invalid peer certificate: UnknownIssuer"),
      "untrusted",
    ],
    ["a Bun refused upgrade", new Error("Expected 101 status code: 404"), "runtime"],
    ["a Deno refused upgrade", networkError("Invalid status code: 502"), "runtime"],
    [
      "a Node 22 failed dial",
      new Error("Received network error or non-101 status code."),
      "unreachable",
    ],
  ])("should map %s", async (_, error, kind) => {
    const err = await streamError(
      failingConnector((socket) => socket.onerror?.({ error })),
    );
    expect(Unreachable.matches(err)).toBe(kind === "unreachable");
    expect(UntrustedCertificate.matches(err)).toBe(kind === "untrusted");
    expect(err).toMatchObject({ cause: error });
  });

  test("should name a possible certificate problem for a failed secure dial", async () => {
    const err = await streamError(
      failingConnector((socket) => socket.onclose?.({ code: 1006 })),
      true,
    );
    expect(Unreachable.matches(err)).toBe(true);
    expect(err).toHaveProperty("message", expect.stringContaining("certificate"));
  });

  test("should not name a certificate for a failed dial without TLS", async () => {
    const err = await streamError(
      failingConnector((socket) => socket.onclose?.({ code: 1006 })),
    );
    expect(err).toHaveProperty("message", "Unreachable");
  });

  test("should throw Unreachable when the socket closes before it opens", async () => {
    const err = await streamError(
      failingConnector((socket) => socket.onclose?.({ code: 1006 })),
    );
    expect(Unreachable.matches(err)).toBe(true);
  });

  test("should throw Unreachable when the dial returns a closed socket", async () => {
    const err = await streamError(
      failingConnector(() => {}, globalThis.WebSocket.CLOSED),
    );
    expect(Unreachable.matches(err)).toBe(true);
  });

  test("should throw Unreachable when the server never answers the dial", async () => {
    const sockets: NetSocket[] = [];
    const server = createServer((socket) => sockets.push(socket));
    await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
    const { port } = server.address() as AddressInfo;
    const c = new WebSocketClient({
      endpoint: new url.URL({ host: "127.0.0.1", port }),
      codec: binary.JSON_CODEC,
      openTimeout: TimeSpan.milliseconds(100),
    });
    const err = await c.stream("stream", messageSchema, messageSchema).then(
      () => null,
      (e: unknown) => e,
    );
    expect(Unreachable.matches(err)).toBe(true);
    expect(err).toHaveProperty("message", "stream did not open within 100ms");
    expect(err).toHaveProperty("cause", expect.any(Timeout));
    sockets.forEach((socket) => socket.destroy());
    await new Promise((resolve) => server.close(resolve));
  });

  test("should throw Unreachable when the server never acks the open", async () => {
    const server = new WebSocketServer({ port: 0, host: "127.0.0.1" });
    await new Promise((resolve) => server.on("listening", resolve));
    const closed = new Promise((resolve) =>
      server.on("connection", (ws) => ws.on("close", resolve)),
    );
    const { port } = server.address() as AddressInfo;
    const c = new WebSocketClient({
      endpoint: new url.URL({ host: "127.0.0.1", port }),
      codec: binary.JSON_CODEC,
      openTimeout: TimeSpan.milliseconds(100),
    });
    await expect(c.stream("stream", messageSchema, messageSchema)).rejects.toThrow(
      Unreachable,
    );
    await closed;
    await new Promise((resolve) => server.close(resolve));
  });
});

describe("local server", () => {
  interface Connection {
    ws: WebSocket;
    req: IncomingMessage;
  }

  let server: WebSocketServer;
  let connected: Promise<Connection>;
  let sendOpenAck: (ws: WebSocket) => void;

  const dial = async (
    flowControl: Partial<FlowControl>,
    codec: binary.Codec = new binary.JSONCodec(),
    keepAlive: Partial<KeepAlive> = {},
  ) => {
    const { port } = server.address() as AddressInfo;
    const c = new WebSocketClient({
      endpoint: new url.URL({ host: "127.0.0.1", port }),
      codec,
      flowControl,
      keepAlive,
    });
    return await c.stream("stream", messageSchema, messageSchema);
  };

  const sendData = (ws: WebSocket, payload: z.infer<typeof messageSchema>) =>
    ws.send(Buffer.from(JSON.stringify({ type: "data", payload })));

  const OPEN_ACK = Buffer.from(JSON.stringify({ type: "open", payload: null }));
  const KEEP_ALIVE_MESSAGE = '{"type":"keep_alive"}';

  beforeEach(async () => {
    server = new WebSocketServer({ port: 0, host: "127.0.0.1" });
    sendOpenAck = (ws) => ws.send(OPEN_ACK);
    connected = new Promise((resolve) =>
      server.on("connection", (ws, req) => {
        sendOpenAck(ws);
        resolve({ ws, req });
      }),
    );
    await new Promise((resolve) => server.on("listening", resolve));
  });

  afterEach(async () => {
    server.clients.forEach((ws) => ws.terminate());
    await new Promise((resolve) => server.close(resolve));
  });

  test("should reject a payload that does not match the schema", async () => {
    const stream = await dial({});
    const { ws } = await connected;
    ws.send(Buffer.from(JSON.stringify({ type: "data", payload: { id: "a" } })));
    await expect(stream.receive()).rejects.toThrow(/stream response/);
  });

  test("should return the payload a codec builds without parsing it", async () => {
    // Builds payloads without a schema, as the binary frame codecs do.
    const codec: binary.Codec = {
      contentType: "application/json",
      encode: (payload) => binary.JSON_CODEC.encode(payload),
      decode: <P extends z.ZodType>(data: Uint8Array | ArrayBuffer) =>
        binary.JSON_CODEC.decode<P>(data),
    };
    const stream = await dial({}, codec);
    const { ws } = await connected;
    ws.send(Buffer.from(JSON.stringify({ type: "data", payload: { id: "a" } })));
    expect(await stream.receive()).toEqual({ id: "a" });
  });

  test("should receive queued messages in order", async () => {
    const stream = await dial({});
    const { ws } = await connected;
    for (let id = 0; id < 10; id++) sendData(ws, { id });
    for (let id = 0; id < 10; id++) expect((await stream.receive()).id).toEqual(id);
  });

  test("should reject receive when a message does not decode", async () => {
    const stream = await dial({});
    const { ws } = await connected;
    ws.send(Buffer.from("not json"));
    await expect(stream.receive()).rejects.toThrow(SyntaxError);
  });

  test("should keep the flow control limits in a client from withCodec", async () => {
    const { port } = server.address() as AddressInfo;
    const c = new WebSocketClient({
      endpoint: new url.URL({ host: "127.0.0.1", port }),
      codec: new binary.JSONCodec(),
      flowControl: { receiveQueueLimit: 1024 },
    }).withCodec(new binary.JSONCodec());
    const stream = await c.stream("stream", messageSchema, messageSchema);
    const { ws } = await connected;
    const closed = new Promise((resolve) => ws.on("close", resolve));
    for (let id = 0; id < 10; id++) sendData(ws, { id, message: "x".repeat(256) });
    await closed;
    await expect(stream.receive()).rejects.toThrow(ReceiveOverflow);
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

  test("should throw EOF on send after the server closes the socket", async () => {
    const stream = await dial({});
    const { ws } = await connected;
    const closed = new Promise((resolve) => ws.on("close", resolve));
    ws.close();
    await closed;
    await expect(stream.send({ id: 0 })).rejects.toThrow(EOF);
  });

  test("should open a stream with no open timeout", async () => {
    const { port } = server.address() as AddressInfo;
    const c = new WebSocketClient({
      endpoint: new url.URL({ host: "127.0.0.1", port }),
      codec: new binary.JSONCodec(),
      openTimeout: TimeSpan.ZERO,
    });
    const stream = await c.stream("stream", messageSchema, messageSchema);
    const { ws } = await connected;
    sendData(ws, { id: 1 });
    expect((await stream.receive()).id).toEqual(1);
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

  test("should send waiting requests in call order", async () => {
    const stream = await dial({ sendHighWaterMark: 1024 });
    const { ws, req } = await connected;
    req.socket.pause();
    const received: number[] = [];
    const codec = new binary.JSONCodec();
    const msgZ = z.object({ payload: messageSchema });
    ws.on("message", (data: Buffer) => {
      received.push(codec.decode(data, msgZ).payload.id ?? -1);
    });
    const first = new Promise((resolve) => ws.once("message", resolve));
    await stream.send({ id: 0, message: "x".repeat(32 * 1024 * 1024) });
    // Holds the drain poll of the waiting send, so the buffer drains before it wakes.
    vi.useFakeTimers({ toFake: ["setTimeout"] });
    try {
      const waiting = stream.send({ id: 1 });
      req.socket.resume();
      await first;
      const late = stream.send({ id: 2 });
      await vi.advanceTimersByTimeAsync(100);
      await Promise.all([waiting, late]);
    } finally {
      vi.useRealTimers();
    }
    await expect.poll(() => received).toEqual([0, 1, 2]);
  });

  describe("keep-alive", () => {
    const TIMEOUT = TimeSpan.milliseconds(200);
    const keepAlive = { interval: TimeSpan.milliseconds(50), timeout: TIMEOUT };

    const createClient = (config: Partial<KeepAlive>): WebSocketClient =>
      new WebSocketClient({ endpoint, codec: binary.JSON_CODEC, keepAlive: config });

    const sendKeepAliveFirst = (ws: WebSocket): void => {
      ws.send(KEEP_ALIVE_MESSAGE);
      ws.send(OPEN_ACK);
    };

    test("should ask for the interval in the URL", async () => {
      await dial({}, undefined, { interval: TimeSpan.seconds(2) });
      const { req } = await connected;
      expect(req.url).toContain("freighterctxkeep_alive=2000000000");
    });

    test("should default the timeout to three intervals", async () => {
      sendOpenAck = sendKeepAliveFirst;
      const stream = await dial({}, undefined, {
        interval: TimeSpan.milliseconds(100),
      });
      const started = performance.now();
      await expect(stream.receive()).rejects.toThrow(
        "stream received no message for 300ms",
      );
      expect(performance.now() - started).toBeGreaterThanOrEqual(250);
    });

    test("should reject a timeout that is not longer than the interval", () => {
      expect(() =>
        createClient({ interval: TimeSpan.seconds(20), timeout: TimeSpan.seconds(15) }),
      ).toThrow("keep-alive timeout 15s must be longer than the interval 20s");
    });

    test("should reject a negative interval", () => {
      expect(() => createClient({ interval: TimeSpan.seconds(-1) })).toThrow(
        "keep-alive interval -1s is negative",
      );
    });

    test("should not fail a stream when the timeout sets no limit", async () => {
      sendOpenAck = sendKeepAliveFirst;
      const stream = await dial({}, undefined, {
        interval: TimeSpan.milliseconds(50),
        timeout: Infinity,
      });
      const { ws } = await connected;
      await sleep.sleep(TimeSpan.milliseconds(200));
      sendData(ws, { id: 1 });
      expect(await stream.receive()).toEqual({ id: 1 });
    });

    test("should not ask for keep-alives with a zero interval", async () => {
      await dial({}, undefined, { interval: TimeSpan.ZERO });
      const { req } = await connected;
      expect(req.url).not.toContain("keep_alive");
    });

    test("should fail a stalled stream with Unreachable", async () => {
      sendOpenAck = sendKeepAliveFirst;
      const stream = await dial({}, undefined, keepAlive);
      const started = performance.now();
      await expect(stream.receive()).rejects.toSatisfy(
        (err) =>
          Unreachable.matches(err) &&
          err.message === `stream received no message for ${TIMEOUT.toString()}` &&
          err.url.port === (server.address() as AddressInfo).port,
      );
      expect(performance.now() - started).toBeGreaterThanOrEqual(
        TIMEOUT.milliseconds - 50,
      );
      await expect(stream.send({ id: 0 })).rejects.toThrow(EOF);
    });

    test("should keep a stream open while keep-alives arrive", async () => {
      sendOpenAck = sendKeepAliveFirst;
      const stream = await dial({}, undefined, keepAlive);
      const { ws } = await connected;
      const ticker = setInterval(() => ws.send(KEEP_ALIVE_MESSAGE), 50);
      try {
        await sleep.sleep(TIMEOUT.mult(3));
        sendData(ws, { id: 1 });
        expect(await stream.receive()).toEqual({ id: 1 });
        expect(stream.received()).toBe(false);
      } finally {
        clearInterval(ticker);
      }
    });

    test("should not fail a stream when the server sends no keep-alives", async () => {
      const stream = await dial({}, undefined, keepAlive);
      const { ws } = await connected;
      await sleep.sleep(TIMEOUT.mult(2));
      sendData(ws, { id: 1 });
      expect(await stream.receive()).toEqual({ id: 1 });
    });

    test("should fail the stream on a text message that is not a keep-alive", async () => {
      sendOpenAck = sendKeepAliveFirst;
      const stream = await dial({}, undefined, keepAlive);
      const { ws } = await connected;
      ws.send("hello");
      await expect(stream.receive()).rejects.toThrow("unexpected text message: hello");
    });
  });

  describe("authorization", () => {
    const TOKEN = "Bearer test-token";

    const authorize = async (connector?: Connector) => {
      const { port } = server.address() as AddressInfo;
      const c = new WebSocketClient({
        endpoint: new url.URL({ host: "127.0.0.1", port }),
        codec: new binary.JSONCodec(),
        connector,
      });
      c.use(async (ctx, next) => {
        ctx.params.Authorization = TOKEN;
        return await next(ctx);
      });
      await c.stream("stream", messageSchema, messageSchema);
      return (await connected).req;
    };

    test("should send the token in a header and not in the URL", async () => {
      const req = await authorize();
      expect(req.headers.authorization).toEqual(TOKEN);
      expect(req.url).not.toContain("test-token");
      expect(req.url).toContain("freighterctxContent-Type=application%2Fjson");
    });

    describe("connector that cannot send headers", () => {
      test("should not put the token in the URL", async () => {
        const req = await authorize({
          fetch: async () => await Promise.reject(new Error("not used")),
          dial: async (u) => new globalThis.WebSocket(u),
        });
        expect(req.url).not.toContain("test-token");
      });

      test("should send the token in a subprotocol", async () => {
        const req = await authorize({
          fetch: async () => await Promise.reject(new Error("not used")),
          dial: async (u, headers) =>
            new globalThis.WebSocket(u, subprotocols(headers.Authorization)),
        });
        expect(req.headers["sec-websocket-protocol"]).toEqual(
          "synnax.v1, synnax.token.test-token",
        );
      });
    });

    describe("DOM WebSocket", () => {
      beforeEach(() => vi.stubGlobal("document", {}));
      afterEach(() => vi.unstubAllGlobals());

      test("should send the token in a subprotocol and not in the URL", async () => {
        const req = await authorize();
        expect(req.headers["sec-websocket-protocol"]).toEqual(
          "synnax.v1, synnax.token.test-token",
        );
        expect(req.headers.authorization).toBeUndefined();
        expect(req.url).not.toContain("test-token");
      });
    });
  });
});
