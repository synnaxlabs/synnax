import { ZodSchema, z } from 'zod';

import { EncoderDecoder } from './encoder';
import { EOF, ErrorPayloadSchema, StreamClosed, decodeError } from './errors';
import { buildQueryString } from './http';
import { CONTENT_TYPE_HEADER_KEY } from './http';
import { MetaData, MiddlewareCollector } from './middleware';
import { RUNTIME, Runtime } from './runtime';
import { Stream, StreamClient } from './stream';
import URL from './url';

const resolveWebSocketConstructor = (): typeof WebSocket => {
  if (RUNTIME == Runtime.Node) return require('ws');
  return WebSocket;
};

enum MessageType {
  Data = 'data',
  Close = 'close',
}

const MessageSchema = z.object({
  type: z.nativeEnum(MessageType),
  payload: z.unknown().optional(),
  error: z.optional(ErrorPayloadSchema),
});

type Message = z.infer<typeof MessageSchema>;

enum CloseCode {
  Normal = 1000,
  GoingAway = 1001,
}

type ReceiveCallbacksQueue = {
  resolve: (msg: Message) => void;
  reject: (reason: unknown) => void;
}[];

/** WebSocketStream is an implementation of Stream that is backed by a websocket. */
class WebSocketStream<RQ, RS> implements Stream<RQ, RS> {
  private encoder: EncoderDecoder;
  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
  // @ts-ignore
  private reqSchema: z.ZodSchema<RQ>;
  private resSchema: z.ZodSchema<RS>;

  private ws: WebSocket;
  private server_closed?: Error;
  private send_closed: boolean;
  private receiveDataQueue: Message[] = [];
  private receiveCallbacksQueue: ReceiveCallbacksQueue = [];

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
    this.send_closed = false;
    this.listenForMessages();
  }

  /** Implements the Stream protocol */
  send(req: RQ): Error | undefined {
    if (this.server_closed) return new EOF();
    if (this.send_closed) throw new StreamClosed();

    this.ws.send(
      this.encoder.encode({
        type: MessageType.Data,
        payload: req,
        error: undefined,
      })
    );

    return undefined;
  }

  /** Implements the Stream protocol */
  async receive(): Promise<[RS | undefined, Error | undefined]> {
    if (this.server_closed) return [undefined, this.server_closed];

    const msg = await this.receiveMsg();

    if (msg.type == MessageType.Close) {
      if (!msg.error) throw new Error('Message error must be defined');
      this.server_closed = decodeError(msg.error);
      return [undefined, this.server_closed];
    }

    return [this.resSchema.parse(msg.payload), undefined];
  }

  /** Implements the Stream protocol */
  received(): boolean {
    return this.receiveDataQueue.length !== 0;
  }

  /** Implements the Stream protocol */
  closeSend(): void {
    if (this.send_closed || this.server_closed) {
      return undefined;
    }
    const msg: Message = { type: MessageType.Close };
    try {
      this.ws.send(this.encoder.encode(msg));
    } finally {
      this.send_closed = true;
    }
    return undefined;
  }

  private async receiveMsg(): Promise<Message> {
    if (this.receiveDataQueue.length !== 0) {
      const msg = this.receiveDataQueue.shift();
      if (msg) return msg;
      throw new Error('unexpected undefined message');
    }

    return new Promise((resolve, reject) => {
      this.receiveCallbacksQueue.push({ resolve, reject });
    });
  }

  private listenForMessages(): void {
    this.ws.onmessage = (ev: MessageEvent) => {
      const msg = this.encoder.decode(ev.data, MessageSchema);

      if (this.receiveCallbacksQueue.length > 0) {
        const callback = this.receiveCallbacksQueue.shift();
        if (callback) callback.resolve(msg);
        else throw new Error('Unexpected empty callback queue');
      } else {
        this.receiveDataQueue.push(msg);
      }
    };

    this.ws.onclose = (ev: CloseEvent) => {
      if ([CloseCode.Normal, CloseCode.GoingAway].includes(ev.code)) {
        this.server_closed = new EOF();
      } else {
        this.server_closed = new StreamClosed();
      }
    };
  }
}

export const FREIGHTER_METADATA_PREFIX = 'freightermd';

/**
 * WebSocketClient is an implementation of StreamClient that is backed by
 * websockets.
 */
export class WebSocketClient
  extends MiddlewareCollector
  implements StreamClient
{
  baseUrl: URL;
  encoder: EncoderDecoder;

  static readonly MESSAGE_TYPE = 'arraybuffer';

  /**
   * @param encoder - The encoder to use for encoding messages and decoding
   *   responses.
   * @param baseURL - A base url to use as a prefix for all requests.
   */
  constructor(encoder: EncoderDecoder, baseURL: URL) {
    super();
    this.baseUrl = baseURL.replace({ protocol: 'ws' });
    this.encoder = encoder;
  }

  /** Implements the StreamClient interface. */
  async stream<RQ, RS>(
    target: string,
    reqSchema: ZodSchema<RQ>,
    resSchema: ZodSchema<RS>
  ): Promise<Stream<RQ, RS>> {
    const socketConstructor = resolveWebSocketConstructor();
    let stream: Stream<RQ, RS> | undefined;
    const error = await this.executeMiddleware(
      { target, protocol: 'websocket', params: {} },
      async (md: MetaData): Promise<Error | undefined> => {
        const ws = new socketConstructor(this.buildURL(target, md));
        ws.binaryType = WebSocketClient.MESSAGE_TYPE;
        const streamOrErr = await this.wrapSocket(ws, reqSchema, resSchema);
        if (streamOrErr instanceof Error) return streamOrErr;
        stream = streamOrErr;
        return undefined;
      }
    );
    if (error) throw error;
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
    return this.baseUrl.child(target).stringify() + qs;
  }

  private async wrapSocket<RQ, RS>(
    ws: WebSocket,
    reqSchema: ZodSchema<RQ>,
    resSchema: ZodSchema<RS>
  ): Promise<WebSocketStream<RQ, RS> | Error> {
    return await new Promise((resolve, reject) => {
      ws.onopen = () => {
        resolve(
          new WebSocketStream<RQ, RS>(ws, this.encoder, reqSchema, resSchema)
        );
      };
      ws.onerror = (ev: Event) => reject(new Error(ev.toString()));
    });
  }
}
