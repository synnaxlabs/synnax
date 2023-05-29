interface TypedWorkerMessage {
  type: string;
  payload: any;
}

export class RoutedWorker {
  send: Send;
  children: Map<string, TypedWorker<any>>;

  constructor(send: Send, handle: Handle) {
    this.send = send;
    this.children = new Map();
    handle((msg: TypedWorkerMessage) =>
      this.children.get(msg.type)?.handler?.(msg.payload)
    );
  }

  route<RQ, RS = RQ>(type: string): TypedWorker<RQ, RS> {
    const send = typedSend(type, this.send);
    const t = new TypedWorker<RQ, RS>(send);
    this.children.set(type, t);
    return t;
  }
}

type Handle = (callback: (payload: any) => void) => void;
type Handler = (payload: any) => void;
type Send = (payload: any, transfer?: Transferable[]) => void;

const typedSend =
  (type: string, send: Send): Send =>
  (payload: any) =>
    send({ type, payload });

export class TypedWorker<RQ, RS = RQ> {
  private readonly _send: Send;
  handler: Handler | null;

  constructor(send: Send) {
    this._send = send;
    this.handler = null;
  }

  send(payload: RQ, transfer: Transferable[] = []): void {
    this._send(payload, transfer);
  }

  handle(callback: (payload: RS) => void): void {
    this.handler = callback;
  }
}
