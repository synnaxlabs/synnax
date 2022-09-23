import { z, ZodSchema } from 'zod';

import { EncoderDecoder } from './encoder';
import { decodeError, EOF, ErrorPayloadSchema, StreamClosed } from './errors';
import { Runtime, RUNTIME } from './runtime';
import { Stream, StreamClient } from './stream';
import URL from './url';

const resolveWebsocketProvider = (): typeof WebSocket => {
  if (RUNTIME == Runtime.Node) {
    return require('ws');
  }
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

export class WebSocketClientStream<RQ, RS> implements Stream<RQ, RS> {
  private encoder: EncoderDecoder;
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

  send(req: RQ): Error | undefined {
    if (this.server_closed) {
      return new EOF();
    }

    if (this.send_closed) {
      throw new StreamClosed();
    }

    this.ws.send(
      this.encoder.encode({
        type: MessageType.Data,
        payload: this.reqSchema.parse(req),
      })
    );

    return undefined;
  }

  async receive(): Promise<[RS | undefined, Error | undefined]> {
    if (this.server_closed) {
      return [undefined, this.server_closed];
    }

    const msg = await this.receiveMsg();

    if (msg.type == MessageType.Close) {
      if (!msg.error) {
        throw new Error('Message error must be defined');
      }
      this.server_closed = decodeError(msg.error);
      return [undefined, this.server_closed];
    }

    return [this.resSchema.parse(msg.payload), undefined];
  }

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
      if (msg) {
        return msg;
      } else {
        throw new Error('unexpected undefined message');
      }
    }

    return new Promise((resolve, reject) => {
      this.receiveCallbacksQueue.push({ resolve, reject });
    });
  }

  private listenForMessages(): void {
    this.ws.onmessage = (ev: MessageEvent) => {
      const msg = MessageSchema.parse(this.encoder.decode(ev.data));

      if (this.receiveCallbacksQueue.length > 0) {
        const callback = this.receiveCallbacksQueue.shift();
        if (callback) {
          callback.resolve(msg);
        } else {
          throw new Error('Unexpected empty callback queue');
        }
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

export class WebSocketClient implements StreamClient {
  endpoint: URL;
  encoder: EncoderDecoder;

  constructor(baseURL: URL, encoder: EncoderDecoder) {
    this.endpoint = baseURL.child({ protocol: 'ws' });
    this.encoder = encoder;
  }

  async stream<RQ, RS>(
    target: string,
    reqSchema: ZodSchema<RQ>,
    resSchema: ZodSchema<RS>
  ): Promise<Stream<RQ, RS>> {
    const ResolvedWebSocket = resolveWebsocketProvider();
    const url = this.endpoint.path(
      `${target}?contentType=${this.encoder.contentType}`
    );
    const ws = new ResolvedWebSocket(url);
    ws.binaryType = 'arraybuffer';
    return new Promise((resolve, reject) => {
      ws.onopen = () => {
        resolve(
          new WebSocketClientStream<RQ, RS>(
            ws,
            this.encoder,
            reqSchema,
            resSchema
          )
        );
      };
      ws.onerror = (ev: Event) => {
        reject(ev);
      };
    });
  }
}
