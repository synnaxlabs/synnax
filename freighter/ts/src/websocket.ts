// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type binary, buildQueryString, errors, type URL } from "@synnaxlabs/x";
import { z } from "zod";

import { EOF, StreamClosed } from "@/errors";
import { CONTENT_TYPE_HEADER_KEY } from "@/http";
import { type Context, MiddlewareCollector } from "@/middleware";
import { type Stream, type StreamClient } from "@/stream";

const wsMessageZ = z.object({
  type: z.enum(["data", "close", "open"]),
  payload: z.unknown(),
  error: z.optional(errors.payloadZ),
});

export type WebsocketMessage<P = unknown> = {
  type: "data" | "close" | "open";
  error?: errors.Payload;
  payload?: P;
};

type ReceiveCallbacksQueue = Array<{
  resolve: (msg: WebsocketMessage<unknown>) => void;
  reject: (reason: unknown) => void;
}>;

/** WebSocketStream is an implementation of Stream that is backed by a websocket. */
class WebSocketStream<
  RQ extends z.ZodType,
  RS extends z.ZodType = RQ,
> implements Stream<RQ, RS> {
  private readonly codec: binary.Codec;
  private readonly reqSchema: RQ;
  private readonly resSchema: RS;
  private readonly ws: WebSocket;
  private serverClosed: Error | null;
  private sendClosed: boolean;
  private readonly receiveDataQueue: WebsocketMessage[] = [];
  private readonly receiveCallbacksQueue: ReceiveCallbacksQueue = [];

  constructor(ws: WebSocket, encoder: binary.Codec, reqSchema: RQ, resSchema: RS) {
    this.codec = encoder;
    this.reqSchema = reqSchema;
    this.resSchema = resSchema;
    this.ws = ws;
    this.sendClosed = false;
    this.serverClosed = null;
    this.listenForMessages();
  }

  async receiveOpenAck(): Promise<Error | null> {
    const msg = await this.receiveMsg();
    if (msg.type !== "open") {
      if (msg.error == null) throw new Error("Message error must be defined");
      return errors.decode(msg.error);
    }
    return null;
  }

  /** Implements the Stream protocol */
  send(req: z.input<RQ> | z.infer<RQ>): Error | null {
    if (this.serverClosed != null) return new EOF();
    if (this.sendClosed) throw new StreamClosed();
    this.ws.send(this.codec.encode({ type: "data", payload: req }));
    return null;
  }

  /** Implements the Stream protocol */
  async receive(): Promise<[z.infer<RS>, null] | [null, Error]> {
    if (this.serverClosed != null) return [null, this.serverClosed];
    const msg = await this.receiveMsg();
    if (msg.type === "close") {
      if (msg.error == null) throw new Error("Message error must be defined");
      this.serverClosed = errors.decode(msg.error);
      if (this.serverClosed == null) throw new Error("Message error must be defined");
      return [null, this.serverClosed];
    }
    return [this.resSchema.parse(msg.payload), null];
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

  private async receiveMsg(): Promise<WebsocketMessage> {
    const msg = this.receiveDataQueue.shift();
    if (msg != null) return msg;
    return await new Promise((resolve, reject) =>
      this.receiveCallbacksQueue.push({ resolve, reject }),
    );
  }

  private addMessage(msg: WebsocketMessage): void {
    const callback = this.receiveCallbacksQueue.shift();
    if (callback != null) callback.resolve(msg);
    else this.receiveDataQueue.push(msg);
  }

  private listenForMessages(): void {
    this.ws.onmessage = this.onMessage.bind(this);
    this.ws.onclose = this.onClose.bind(this);
  }

  private onMessage(ev: MessageEvent<Uint8Array>): void {
    this.addMessage(this.codec.decode(ev.data, wsMessageZ));
  }

  private onClose(ev: CloseEvent): void {
    this.addMessage({
      type: "close",
      error: {
        type: ev.code === CLOSE_NORMAL ? EOF.TYPE : StreamClosed.TYPE,
        data: "",
      },
    });
  }
}

export const FREIGHTER_METADATA_PREFIX = "freighterctx";

const CLOSE_NORMAL = 1000;

/**
 * WebSocketClient is an implementation of StreamClient that is backed by
 * websockets.
 */
export class WebSocketClient extends MiddlewareCollector implements StreamClient {
  baseUrl: URL;
  encoder: binary.Codec;
  secure: boolean;

  static readonly MESSAGE_TYPE = "arraybuffer";

  /**
   * @param encoder - The encoder to use for encoding messages and decoding
   *   responses.
   * @param baseEndpoint - A base url to use as a prefix for all requests.
   */
  constructor(baseEndpoint: URL, encoder: binary.Codec, secure = false) {
    super();
    this.secure = secure;
    this.baseUrl = baseEndpoint.replace({ protocol: secure ? "wss" : "ws" });
    this.encoder = encoder;
  }

  withCodec(codec: binary.Codec): WebSocketClient {
    const c = new WebSocketClient(this.baseUrl, codec, this.secure);
    c.use(...this.middleware);
    return c;
  }

  /** Implements the StreamClient interface. */
  async stream<RQ extends z.ZodType, RS extends z.ZodType = RQ>(
    target: string,
    reqSchema: RQ,
    resSchema: RS,
  ): Promise<Stream<RQ, RS>> {
    let stream: Stream<RQ, RS> | undefined;
    const [, error] = await this.executeMiddleware(
      { target, protocol: "websocket", params: {}, role: "client" },
      async (ctx: Context): Promise<[Context, Error | null]> => {
        const ws = new WebSocket(this.buildURL(target, ctx));
        const outCtx: Context = { ...ctx, params: {} };
        ws.binaryType = WebSocketClient.MESSAGE_TYPE;
        const streamOrErr = await this.wrapSocket(ws, reqSchema, resSchema);
        if (streamOrErr instanceof Error) return [outCtx, streamOrErr];
        stream = streamOrErr;
        return [outCtx, null];
      },
    );
    if (error != null) throw error;
    return stream as Stream<RQ, RS>;
  }

  private buildURL(target: string, ctx: Context): string {
    const qs = buildQueryString(
      {
        [CONTENT_TYPE_HEADER_KEY]: this.encoder.contentType,
        ...ctx.params,
      },
      FREIGHTER_METADATA_PREFIX,
    );
    return this.baseUrl.child(target).toString() + qs;
  }

  private async wrapSocket<RQ extends z.ZodType, RS extends z.ZodType = RQ>(
    ws: WebSocket,
    reqSchema: RQ,
    resSchema: RS,
  ): Promise<WebSocketStream<RQ, RS> | Error> {
    return await new Promise((resolve) => {
      ws.onopen = () => {
        const oWs = new WebSocketStream<RQ, RS>(ws, this.encoder, reqSchema, resSchema);
        oWs
          .receiveOpenAck()
          .then((err) => {
            if (err != null) resolve(err);
            else resolve(oWs);
          })
          .catch((err: Error) => resolve(err));
      };
      ws.onerror = (ev: Event) => {
        const ev_ = ev as ErrorEvent;
        resolve(new Error(ev_.message));
      };
    });
  }
}
