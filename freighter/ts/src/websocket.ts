// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type binary, errors, sleep, TimeSpan, url } from "@synnaxlabs/x";
import { z } from "zod";

import { type Connector, DEFAULT_CONNECTOR, type Socket } from "@/connector";
import { EOF, ReceiveOverflow, StreamClosed, Unreachable } from "@/errors";
import { CONTENT_TYPE_HEADER_KEY, FREIGHTER_METADATA_PREFIX } from "@/http";
import { type Context, MiddlewareCollector } from "@/middleware";
import type { Stream, StreamClient } from "@/stream";

const controlMessageZ = z.object({
  type: z.enum(["close", "open"]),
  error: z.optional(errors.payloadZ),
});

type ControlMessage = z.infer<typeof controlMessageZ>;

type Message<P> = { type: "data"; payload: P } | ControlMessage;

const wireMessageZ = z.discriminatedUnion("type", [
  z.object({ type: z.literal("data"), payload: z.unknown() }),
  controlMessageZ,
]);

// The codec applies this schema, so a codec that builds typed payloads itself skips
// parsing them.
const messageZ = <RS extends z.ZodType>(payload: RS) =>
  wireMessageZ.transform((msg, ctx): Message<z.infer<RS>> => {
    if (msg.type !== "data") return msg;
    const res = payload.safeParse(msg.payload);
    if (res.success) return { type: "data", payload: res.data };
    for (const issue of res.error.issues)
      ctx.addIssue({ ...issue, path: ["payload", ...issue.path] });
    return z.NEVER;
  });

export type WebsocketMessage<P = unknown> = {
  type: "data" | "close" | "open";
  error?: errors.Payload;
  payload?: P;
};

/** Limits that keep a WebSocket stream from buffering without bound. */
export interface FlowControl {
  /** Bytes the socket may buffer before send waits for them to go out. */
  sendHighWaterMark: number;
  /** Bytes of unread messages a stream may hold before it fails. */
  receiveQueueLimit: number;
}

const DEFAULT_FLOW_CONTROL: FlowControl = {
  sendHighWaterMark: 8 * 1024 * 1024,
  receiveQueueLimit: 256 * 1024 * 1024,
};

// WebSocket has no drain event, so send polls bufferedAmount.
const DRAIN_POLL_INTERVAL = TimeSpan.milliseconds(5);

// Data messages stay raw until receive decodes them.
type QueuedMessage = ArrayBuffer | ControlMessage;

type ReceiveCallbacksQueue = Array<{
  resolve: (msg: QueuedMessage) => void;
  reject: (reason: unknown) => void;
}>;

/** WebSocketStream is an implementation of Stream that is backed by a websocket. */
class WebSocketStream<
  RQ extends z.ZodType,
  RS extends z.ZodType = RQ,
> implements Stream<RQ, RS> {
  private readonly codec: binary.Codec;
  private readonly reqSchema: RQ;
  private readonly messageZ: ReturnType<typeof messageZ<RS>>;
  private readonly ws: Socket;
  private serverClosed: Error | null;
  private sendClosed: boolean;
  private readonly flowControl: FlowControl;
  private sendTurn: Promise<void> = Promise.resolve();
  private waitingSends = 0;
  private readonly receiveDataQueue: QueuedMessage[] = [];
  private receiveQueueBytes = 0;
  private readonly receiveCallbacksQueue: ReceiveCallbacksQueue = [];
  private readonly resLabel: string;

  constructor(
    ws: Socket,
    encoder: binary.Codec,
    target: string,
    reqSchema: RQ,
    resSchema: RS,
    flowControl: FlowControl,
  ) {
    this.codec = encoder;
    this.flowControl = flowControl;
    this.resLabel = `${target} response`;
    this.reqSchema = reqSchema;
    this.messageZ = messageZ(resSchema);
    this.ws = ws;
    this.sendClosed = false;
    this.serverClosed = null;
    this.listenForMessages();
  }

  async receiveOpenAck(): Promise<void> {
    const msg = await this.receiveMsg();
    if (msg.type === "open") return;
    if (msg.type === "data") throw new Error("Unexpected open-ack message type: data");
    if (msg.error == null) throw new Error("Message error must be defined");
    const err = errors.decode(msg.error);
    throw err ?? new Error(`Unexpected open-ack message type: ${msg.type}`);
  }

  /** Implements the Stream protocol */
  async send(req: z.input<RQ> | z.infer<RQ>): Promise<void> {
    if (this.waitingSends === 0 && !this.sendBufferFull()) return this.write(req);
    // Waiting sends take turns on one chain, so they go out in call order.
    this.waitingSends++;
    const turn = this.sendTurn.then(async () => await this.drain());
    this.sendTurn = turn;
    try {
      await turn;
    } finally {
      this.waitingSends--;
    }
    this.write(req);
  }

  private sendBufferFull(): boolean {
    return (
      this.ws.bufferedAmount > this.flowControl.sendHighWaterMark &&
      this.ws.readyState === WebSocket.OPEN
    );
  }

  private async drain(): Promise<void> {
    while (this.sendBufferFull()) await sleep.sleep(DRAIN_POLL_INTERVAL);
  }

  private write(req: z.input<RQ> | z.infer<RQ>): void {
    if (this.serverClosed != null) throw new EOF();
    if (this.sendClosed) throw new StreamClosed();
    this.ws.send(this.codec.encode({ type: "data", payload: req }));
  }

  /** Implements the Stream protocol */
  async receive(): Promise<z.infer<RS>> {
    if (this.serverClosed != null) throw this.serverClosed;
    const msg = await this.receiveMsg();
    if (msg.type === "data") return msg.payload;
    if (msg.type === "open") throw new Error("Unexpected open message");
    if (msg.error == null) throw new Error("Message error must be defined");
    this.serverClosed = errors.decode(msg.error);
    if (this.serverClosed == null) throw new Error("Message error must be defined");
    throw this.serverClosed;
  }

  /** Implements the Stream protocol */
  received(): boolean {
    return this.receiveDataQueue.length !== 0;
  }

  /** Implements the Stream protocol */
  closeSend(): void {
    if (this.sendClosed || this.serverClosed != null) return undefined;
    const msg: WebsocketMessage = { type: "close" };
    try {
      this.ws.send(this.codec.encode(msg));
    } finally {
      this.sendClosed = true;
    }
    return undefined;
  }

  private async receiveMsg(): Promise<Message<z.infer<RS>>> {
    const queued = this.receiveDataQueue.shift();
    if (queued != null && !("type" in queued))
      this.receiveQueueBytes -= queued.byteLength;
    const msg =
      queued ??
      (await new Promise<QueuedMessage>((resolve, reject) =>
        this.receiveCallbacksQueue.push({ resolve, reject }),
      ));
    if ("type" in msg) return msg;
    return this.codec.decode(msg, this.messageZ, { label: this.resLabel });
  }

  private addMessage(msg: QueuedMessage): void {
    const callback = this.receiveCallbacksQueue.shift();
    if (callback != null) return callback.resolve(msg);
    this.receiveDataQueue.push(msg);
    if ("type" in msg) return;
    this.receiveQueueBytes += msg.byteLength;
    if (this.receiveQueueBytes > this.flowControl.receiveQueueLimit) this.overflow();
  }

  private overflow(): void {
    const limit = this.flowControl.receiveQueueLimit;
    this.ws.onmessage = null;
    this.ws.onclose = null;
    this.ws.close();
    this.receiveDataQueue.length = 0;
    this.receiveQueueBytes = 0;
    this.serverClosed = new ReceiveOverflow(
      `stream holds more than ${limit} bytes of unread messages`,
    );
    this.addMessage({ type: "close", error: errors.encode(this.serverClosed) });
  }

  private listenForMessages(): void {
    this.ws.onmessage = this.onMessage.bind(this);
    this.ws.onclose = this.onClose.bind(this);
  }

  private onMessage(ev: { data: ArrayBuffer }): void {
    this.addMessage(ev.data);
  }

  private onClose(ev: { code: number }): void {
    this.addMessage({
      type: "close",
      error: {
        type: ev.code === CLOSE_NORMAL ? EOF.TYPE : StreamClosed.TYPE,
        data: "",
      },
    });
  }
}

const CLOSE_NORMAL = 1000;

// The DOM, undici, Deno, and Bun error events carry the error in different fields.
const eventCause = (ev: object): unknown =>
  "error" in ev && ev.error != null ? ev.error : ev;

const eventMessage = (ev: object): string =>
  "message" in ev && typeof ev.message === "string" ? ev.message : "websocket error";

const eventError = (ev: object): Error =>
  new Error(eventMessage(ev), { cause: eventCause(ev) });

// Bun and Deno name a failure from a server that answered: a TLS failure, or an upgrade
// refused with an HTTP status. Node and browsers give no cause for a failed dial, so
// these failures there become Unreachable.
const ANSWERED_CODE = /CERT|SIGNATURE|^ERR_TLS_|^ERR_SSL_/;
const ANSWERED_MESSAGE = /TLS handshake|invalid peer certificate|status code/;

const isAnswered = (cause: unknown): boolean => {
  if (typeof cause !== "object" || cause === null) return false;
  if (
    "code" in cause &&
    typeof cause.code === "string" &&
    ANSWERED_CODE.test(cause.code)
  )
    return true;
  return (
    "message" in cause &&
    typeof cause.message === "string" &&
    ANSWERED_MESSAGE.test(cause.message)
  );
};

const AUTHORIZATION_HEADER_KEY = "Authorization";

/** Configures a {@link WebSocketClient}. */
export interface WebSocketClientConfig {
  /** A base URL to use as a prefix for all requests. */
  endpoint: url.URL;
  /** Encodes messages and decodes responses. */
  codec: binary.Codec;
  /** Whether to connect over WSS. Defaults to false. */
  secure?: boolean;
  /** Overrides for the default flow control limits. */
  flowControl?: Partial<FlowControl>;
  /** Opens the WebSockets. Defaults to the global WebSocket. */
  connector?: Connector;
}

/**
 * WebSocketClient is an implementation of StreamClient that is backed by websockets.
 */
export class WebSocketClient extends MiddlewareCollector implements StreamClient {
  baseUrl: url.URL;
  encoder: binary.Codec;
  secure: boolean;
  flowControl: FlowControl;
  private readonly connector: Connector;

  static readonly MESSAGE_TYPE = "arraybuffer";

  constructor({
    endpoint,
    codec,
    secure = false,
    flowControl = {},
    connector = DEFAULT_CONNECTOR,
  }: WebSocketClientConfig) {
    super();
    this.secure = secure;
    this.flowControl = { ...DEFAULT_FLOW_CONTROL, ...flowControl };
    this.baseUrl = endpoint.replace({ protocol: secure ? "wss" : "ws" });
    this.encoder = codec;
    this.connector = connector;
  }

  withCodec(codec: binary.Codec): WebSocketClient {
    const c = new WebSocketClient({
      endpoint: this.baseUrl,
      codec,
      secure: this.secure,
      flowControl: this.flowControl,
      connector: this.connector,
    });
    c.use(...this.middleware);
    return c;
  }

  /**
   * Implements the StreamClient interface.
   * @throws {Unreachable} if the socket fails or closes before it opens. A TLS failure
   * or a refused upgrade that the runtime names (Bun and Deno) throws the runtime
   * error instead.
   */
  async stream<RQ extends z.ZodType, RS extends z.ZodType = RQ>(
    target: string,
    reqSchema: RQ,
    resSchema: RS,
  ): Promise<Stream<RQ, RS>> {
    let stream: Stream<RQ, RS> | undefined;
    await this.executeMiddleware(
      { target, protocol: "websocket", params: {}, role: "client" },
      async (ctx: Context): Promise<Context> => {
        const { [AUTHORIZATION_HEADER_KEY]: authorization, ...params } = ctx.params;
        const ws =
          authorization == null || this.connector.sendsHeaders !== true
            ? await this.connector.dial(this.buildURL(target, ctx.params), {})
            : await this.connector.dial(this.buildURL(target, params), {
                [AUTHORIZATION_HEADER_KEY]: authorization,
              });
        const outCtx: Context = { ...ctx, params: {} };
        ws.binaryType = WebSocketClient.MESSAGE_TYPE;
        stream = await this.wrapSocket(ws, target, reqSchema, resSchema);
        return outCtx;
      },
    );
    return stream as Stream<RQ, RS>;
  }

  private buildURL(target: string, params: Record<string, string>): string {
    const qs = url.buildQueryString(
      { [CONTENT_TYPE_HEADER_KEY]: this.encoder.contentType, ...params },
      FREIGHTER_METADATA_PREFIX,
    );
    return this.baseUrl.child(target).toString() + qs;
  }

  private async wrapSocket<RQ extends z.ZodType, RS extends z.ZodType = RQ>(
    ws: Socket,
    target: string,
    reqSchema: RQ,
    resSchema: RS,
  ): Promise<WebSocketStream<RQ, RS>> {
    return await new Promise((resolve, reject) => {
      const unreachable = (cause: unknown) =>
        reject(new Unreachable({ url: this.baseUrl.child(target), cause }));
      ws.onclose = unreachable;
      ws.onerror = (ev: object) => {
        const cause = eventCause(ev);
        if (isAnswered(cause)) reject(eventError(ev));
        else unreachable(cause);
      };
      // Node fails a blocked port, such as 1, before the dial returns the socket.
      if (ws.readyState === WebSocket.CLOSED) unreachable(undefined);
      ws.onopen = () => {
        ws.onerror = (ev: object) => reject(eventError(ev));
        const oWs = new WebSocketStream<RQ, RS>(
          ws,
          this.encoder,
          target,
          reqSchema,
          resSchema,
          this.flowControl,
        );
        oWs
          .receiveOpenAck()
          .then(() => resolve(oWs))
          .catch((err: unknown) => reject(errors.fromUnknown(err)));
      };
    });
  }
}
