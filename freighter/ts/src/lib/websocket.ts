// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ZodSchema, z } from "zod";

import type { EncoderDecoder } from "./encoder";
import { EOF, ErrorPayloadSchema, StreamClosed, decodeError } from "./errors";
import { buildQueryString, CONTENT_TYPE_HEADER_KEY } from "./http";
import { MiddlewareCollector } from "./middleware";
import type { MetaData } from "./middleware";
import { RUNTIME } from "./runtime";
import type { Stream, StreamClient } from "./stream";
import URL from "./url";

const resolveWebSocketConstructor = (): typeof WebSocket =>
  RUNTIME === "node" ? require("ws") : WebSocket;

const MessageSchema = z.object({
  type: z.union([z.literal("data"), z.literal("close")]),
  payload: z.unknown().optional(),
  error: z.optional(ErrorPayloadSchema),
});

type Message = z.infer<typeof MessageSchema>;

type ReceiveCallbacksQueue = Array<{
  resolve: (msg: Message) => void;
  reject: (reason: unknown) => void;
}>;

/** WebSocketStream is an implementation of Stream that is backed by a websocket. */
class WebSocketStream<RQ, RS> implements Stream<RQ, RS> {
  private readonly encoder: EncoderDecoder;
  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
  // @ts-expect-error
  private readonly reqSchema: z.ZodSchema<RQ>;
  private readonly resSchema: z.ZodSchema<RS>;
  private readonly ws: WebSocket;
  private serverClosed?: Error;
  private sendClosed: boolean;
  private readonly receiveDataQueue: Message[] = [];
  private readonly receiveCallbacksQueue: ReceiveCallbacksQueue = [];

  constructor(
    ws: WebSocket,
    encoder: EncoderDecoder,
    reqSchema: z.ZodSchema<RQ>,
    resSchema: z.ZodSchema<RS>
  ) {
    this.encoder = encoder;
    this.reqSchema = reqSchema;
    this.resSchema = resSchema;
    this.ws = ws;
    this.sendClosed = false;
    this.listenForMessages();
  }

  /** Implements the Stream protocol */
  send(req: RQ): Error | undefined {
    if (this.serverClosed != null) return new EOF();
    if (this.sendClosed) throw new StreamClosed();
    this.ws.send(this.encoder.encode({ type: "data", payload: req }));
    return undefined;
  }

  /** Implements the Stream protocol */
  async receive(): Promise<[RS | undefined, Error | undefined]> {
    if (this.serverClosed != null) return [undefined, this.serverClosed];
    const msg = await this.receiveMsg();
    if (msg.type === "close") {
      if (msg.error == null) throw new Error("Message error must be defined");
      this.serverClosed = decodeError(msg.error);
      return [undefined, this.serverClosed];
    }
    return [this.resSchema.parse(msg.payload), undefined];
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
    if (this.receiveDataQueue.length > 0) {
      const msg = this.receiveDataQueue.shift();
      if (msg != null) return msg;
      throw new Error("unexpected undefined message");
    }
    return await new Promise((resolve, reject) =>
      this.receiveCallbacksQueue.push({ resolve, reject })
    );
  }

  private listenForMessages(): void {
    this.ws.onmessage = (ev: MessageEvent<Uint8Array>) => {
      const msg = this.encoder.decode(ev.data, MessageSchema);

      if (this.receiveCallbacksQueue.length > 0) {
        const callback = this.receiveCallbacksQueue.shift();
        if (callback != null) callback.resolve(msg);
        else throw new Error("unexpected empty callback queue");
      } else this.receiveDataQueue.push(msg);
    };

    this.ws.onclose = (ev: CloseEvent) => {
      this.serverClosed = isNormalClosure(ev) ? new EOF() : new StreamClosed();
    };
  }
}

export const FREIGHTER_METADATA_PREFIX = "freightermd";

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
  encoder: EncoderDecoder;

  static readonly MESSAGE_TYPE = "arraybuffer";

  /**
   * @param encoder - The encoder to use for encoding messages and decoding
   *   responses.
   * @param baseEndpoint - A base url to use as a prefix for all requests.
   */
  constructor(baseEndpoint: URL, encoder: EncoderDecoder, secure = false) {
    super();
    this.baseUrl = baseEndpoint.replace({ protocol: secure ? "wss" : "ws" });
    this.encoder = encoder;
  }

  /** Implements the StreamClient interface. */
  async stream<RQ, RS>(
    target: string,
    reqSchema: ZodSchema<RQ>,
    resSchema: ZodSchema<RS>
  ): Promise<Stream<RQ, RS>> {
    const SocketConstructor = resolveWebSocketConstructor();
    let stream: Stream<RQ, RS> | undefined;
    const [, error] = await this.executeMiddleware(
      { target, protocol: "websocket", params: {} },
      async (md: MetaData): Promise<[MetaData, Error | undefined]> => {
        const ws = new SocketConstructor(this.buildURL(target, md));
        const outMD: MetaData = { ...md, params: {} };
        ws.binaryType = WebSocketClient.MESSAGE_TYPE;
        const streamOrErr = await this.wrapSocket(ws, reqSchema, resSchema);
        if (streamOrErr instanceof Error) return [outMD, streamOrErr];
        stream = streamOrErr;
        return [outMD, undefined];
      }
    );
    if (error != null) throw error;
    return stream as Stream<RQ, RS>;
  }

  private buildURL(target: string, md: MetaData): string {
    const qs = buildQueryString({
      request: {
        [CONTENT_TYPE_HEADER_KEY]: this.encoder.contentType,
        ...md.params,
      },
      prefix: FREIGHTER_METADATA_PREFIX,
    });
    return this.baseUrl.child(target).toString() + qs;
  }

  private async wrapSocket<RQ, RS>(
    ws: WebSocket,
    reqSchema: ZodSchema<RQ>,
    resSchema: ZodSchema<RS>
  ): Promise<WebSocketStream<RQ, RS> | Error> {
    return await new Promise((resolve) => {
      ws.onopen = () => {
        resolve(new WebSocketStream<RQ, RS>(ws, this.encoder, reqSchema, resSchema));
      };
      // eslint-disable-next-line @typescript-eslint/no-base-to-string
      ws.onerror = (ev: Event) => resolve(new Error(ev.toString()));
    });
  }
}
