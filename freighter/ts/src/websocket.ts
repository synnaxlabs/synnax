// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { runtime, URL, buildQueryString } from "@synnaxlabs/x";
import type { binary } from "@synnaxlabs/x";
import { z } from "zod";

import { EOF, errorZ, StreamClosed, decodeError } from "@/errors";
import { CONTENT_TYPE_HEADER_KEY } from "@/http";
import { MiddlewareCollector, Context } from "@/middleware";
import type { Stream, StreamClient } from "@/stream";

const resolveWebSocketConstructor = (): typeof WebSocket =>
  runtime.RUNTIME === "node" ? require("ws") : WebSocket;

const MessageSchema = z.object({
  type: z.union([z.literal("data"), z.literal("close")]),
  payload: z.unknown().optional(),
  error: z.optional(errorZ),
});

type Message = z.infer<typeof MessageSchema>;

type ReceiveCallbacksQueue = Array<{
  resolve: (msg: Message) => void;
  reject: (reason: unknown) => void;
}>;

/** WebSocketStream is an implementation of Stream that is backed by a websocket. */
class WebSocketStream<RQ extends z.ZodTypeAny, RS extends z.ZodTypeAny = RQ>
  implements Stream<RQ, RS>
{
  private readonly encoder: binary.EncoderDecoder;
  private readonly reqSchema: RQ;
  private readonly resSchema: RS;
  private readonly ws: WebSocket;
  private serverClosed: Error | null;
  private sendClosed: boolean;
  private readonly receiveDataQueue: Message[] = [];
  private readonly receiveCallbacksQueue: ReceiveCallbacksQueue = [];

  constructor(ws: WebSocket, encoder: binary.EncoderDecoder, reqSchema: RQ, resSchema: RS) {
    this.encoder = encoder;
    this.reqSchema = reqSchema;
    this.resSchema = resSchema;
    this.ws = ws;
    this.sendClosed = false;
    this.serverClosed = null;
    this.listenForMessages();
  }

  /** Implements the Stream protocol */
  send(req: z.input<RQ>): Error | null {
    if (this.serverClosed != null) return new EOF();
    if (this.sendClosed) throw new StreamClosed();
    this.ws.send(this.encoder.encode({ type: "data", payload: req }));
    return null;
  }

  /** Implements the Stream protocol */
  async receive(): Promise<[z.output<RS> | null, Error | null]> {
    if (this.serverClosed != null) return [null, this.serverClosed];
    const msg = await this.receiveMsg();
    if (msg.type === "close") {
      if (msg.error == null) throw new Error("Message error must be defined");
      this.serverClosed = decodeError(msg.error);
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
    const msg: Message = { type: "close" };
    try {
      this.ws.send(this.encoder.encode(msg));
    } finally {
      this.sendClosed = true;
    }
    return undefined;
  }

  private async receiveMsg(): Promise<Message> {
    const msg = this.receiveDataQueue.shift();
    if (msg != null) return msg;
    return await new Promise((resolve, reject) =>
      this.receiveCallbacksQueue.push({ resolve, reject })
    );
  }

  private listenForMessages(): void {
    this.ws.onmessage = (ev: MessageEvent<Uint8Array>) => {
      const msg = this.encoder.decode(ev.data, MessageSchema);
      const callback = this.receiveCallbacksQueue.shift();
      if (callback != null) callback.resolve(msg);
      else this.receiveDataQueue.push(msg);
    };

    this.ws.onclose = (ev: CloseEvent) => {
      this.serverClosed = isNormalClosure(ev) ? new EOF() : new StreamClosed();
    };
  }
}

export const FREIGHTER_METADATA_PREFIX = "freighterctx";

const CloseNormal = 1000;
const CloseGoingAway = 1001;
const NormalClosures = [CloseNormal, CloseGoingAway];

const isNormalClosure = (ev: CloseEvent): boolean => NormalClosures.includes(ev.code);

/**
 * WebSocketClient is an implementation of StreamClient that is backed by
 * websockets.
 */
export class WebSocketClient extends MiddlewareCollector implements StreamClient {
  baseUrl: URL;
  encoder: binary.EncoderDecoder;

  static readonly MESSAGE_TYPE = "arraybuffer";

  /**
   * @param encoder - The encoder to use for encoding messages and decoding
   *   responses.
   * @param baseEndpoint - A base url to use as a prefix for all requests.
   */
  constructor(baseEndpoint: URL, encoder: binary.EncoderDecoder, secure = false) {
    super();
    this.baseUrl = baseEndpoint.replace({ protocol: secure ? "wss" : "ws" });
    this.encoder = encoder;
  }

  /** Implements the StreamClient interface. */
  async stream<RQ extends z.ZodTypeAny, RS extends z.ZodTypeAny = RQ>(
    target: string,
    reqSchema: RQ,
    resSchema: RS
  ): Promise<Stream<RQ, RS>> {
    const SocketConstructor = resolveWebSocketConstructor();
    let stream: Stream<RQ, RS> | undefined;
    const [, error] = await this.executeMiddleware(
      { target, protocol: "websocket", params: {}, role: "client" },
      async (ctx: Context): Promise<[Context, Error | null]> => {
        const ws = new SocketConstructor(this.buildURL(target, ctx));
        const outCtx: Context = { ...ctx, params: {} };
        ws.binaryType = WebSocketClient.MESSAGE_TYPE;
        const streamOrErr = await this.wrapSocket(ws, reqSchema, resSchema);
        if (streamOrErr instanceof Error) return [outCtx, streamOrErr];
        stream = streamOrErr;
        return [outCtx, null];
      }
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
      FREIGHTER_METADATA_PREFIX
    );
    return this.baseUrl.child(target).toString() + qs;
  }

  private async wrapSocket<RQ extends z.ZodTypeAny, RS extends z.ZodTypeAny = RQ>(
    ws: WebSocket,
    reqSchema: RQ,
    resSchema: RS
  ): Promise<WebSocketStream<RQ, RS> | Error> {
    return await new Promise((resolve) => {
      ws.onopen = () => {
        resolve(new WebSocketStream<RQ, RS>(ws, this.encoder, reqSchema, resSchema));
      };
      ws.onerror = (ev: Event) => {
        const ev_ = ev as ErrorEvent;
        resolve(new Error(ev_.message));
      };
    });
  }
}
