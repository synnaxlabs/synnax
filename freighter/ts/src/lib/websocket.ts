import { z, ZodSchema } from 'zod';

import { EncoderDecoder } from './encoder';
import { decodeError, EOF, ErrorPayloadSchema, StreamClosed } from './errors';
import { Runtime, RUNTIME } from './runtime';
import { Stream, StreamClient } from './stream';
import URL from './url';

const resolveWebsocketProvider = (): typeof WebSocket => {
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

/**
 * WebSocketStream is an implementation of Stream that is backed by a websocket.
 */
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
  private receiveCallbacksQueue: {
    resolve: (msg: Message) => void;
    reject: (reason: unknown) => void;
  }[] = [];

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

/**
 * WebSocketClient is an implementation of StreamClient that is backed by
 * websockets.
 */
export class WebSocketClient implements StreamClient {
  url: URL;
  encoder: EncoderDecoder;

  static readonly MESSAGE_TYPE = 'arraybuffer';

  /**
   * @param encoder - The encoder to use for encoding messages and decoding
   * responses.
   * @param baseURL - A base url to use as a prefix for all requests.
   */
  constructor(encoder: EncoderDecoder, baseURL: URL) {
    this.url = baseURL.replace({ protocol: 'ws' });
    this.encoder = encoder;
  }

  /** Implements the StreamClient interface. */
  async stream<RQ, RS>(
    target: string,
    reqSchema: ZodSchema<RQ>,
    resSchema: ZodSchema<RS>
  ): Promise<Stream<RQ, RS>> {
    const ResolvedWebSocket = resolveWebsocketProvider();
    const url =
      this.url.child(target).stringify() +
      '?contentType=' +
      this.encoder.contentType;
    const ws = new ResolvedWebSocket(url);
    ws.binaryType = WebSocketClient.MESSAGE_TYPE;
    return new Promise((resolve, reject) => {
      ws.onopen = () => {
        resolve(
          new WebSocketStream<RQ, RS>(ws, this.encoder, reqSchema, resSchema)
        );
      };
      ws.onerror = (ev: Event) => {
        reject(ev);
      };
    });
  }
}
