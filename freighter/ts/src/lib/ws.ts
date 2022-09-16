import { EncoderDecoder } from './encoder';
import Endpoint from './endpoint';
import { decodeError, EOF, ErrorPayload, StreamClosed } from './errors';
import { Runtime, RUNTIME } from './runtime';
import { ClientStream, StreamClient } from './stream';
import { Payload } from './transport';

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

type Message<P extends Payload> = {
  type: MessageType;
  payload?: P;
  error?: ErrorPayload;
};

enum CloseCode {
  Normal = 1000,
  GoingAway = 1001,
}

export class WebSocketClientStream<RQ extends Payload, RS extends Payload>
  implements ClientStream<RQ, RS>
{
  private encoder: EncoderDecoder;
  private ws: WebSocket;
  private server_closed?: Error;
  private send_closed: boolean;
  private receiveDataQueue: Message<RS>[] = [];
  private receiveCallbacksQueue: {
    resolve: (msg: Message<RS>) => void;
    reject: (reason: unknown) => void;
  }[] = [];

  constructor(encoder: EncoderDecoder, ws: WebSocket) {
    this.encoder = encoder;
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
        payload: req,
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

    return [msg.payload, undefined];
  }

  closeSend(): void {
    if (this.send_closed || this.server_closed) {
      return undefined;
    }
    const msg: Message<RS> = { type: MessageType.Close };
    try {
      this.ws.send(this.encoder.encode(msg));
    } finally {
      this.send_closed = true;
    }
    return undefined;
  }

  private async receiveMsg(): Promise<Message<RS>> {
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
      const msg = this.encoder.decode<Message<RS>>(ev.data);

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
  endpoint: Endpoint;
  encoder: EncoderDecoder;

  constructor(encoder: EncoderDecoder, endpoint: Endpoint) {
    this.endpoint = endpoint.child({ protocol: 'ws' });
    this.encoder = encoder;
  }

  async stream<RQ extends Payload, RS extends Payload>(
    target: string
  ): Promise<ClientStream<RQ, RS>> {
    const ResolvedWebSocket = resolveWebsocketProvider();
    const url = this.endpoint.path(
      `${target}?contentType=${this.encoder.contentType}`
    );
    const ws = new ResolvedWebSocket(url);
    ws.binaryType = 'arraybuffer';
    return new Promise((resolve, reject) => {
      ws.onopen = () => {
        resolve(new WebSocketClientStream(this.encoder, ws));
      };
      ws.onerror = (ev: Event) => {
        reject(ev);
      };
    });
  }
}
