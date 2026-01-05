// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export interface Sender<T> {
  send: (value: T, transfer?: Transferable[]) => void;
}

export interface Handler<T> {
  handle: (handler: (value: T) => void) => void;
}

export interface SenderHandler<I, O> extends Sender<I>, Handler<O> {}

interface TypedWorkerMessage {
  type: string;
  payload: unknown;
}

type SendFunc = (value: unknown, transfer?: Transferable[]) => void;
type HandlerFunc = (value: unknown) => void;

export class RoutedWorker {
  sender: SendFunc;
  handlers: Map<string, TypedWorker<unknown>>;

  constructor(send: SendFunc) {
    this.sender = send;
    this.handlers = new Map();
  }

  handle({ data }: { data: TypedWorkerMessage }): void {
    const handler = this.handlers.get(data.type)?.handler;
    if (handler == null) console.warn(`No handler for ${data.type}`);
    else handler(data.payload);
  }

  route<RQ, RS = RQ>(type: string): TypedWorker<RQ, RS> {
    const send = typedSend(type, this.sender);
    const t = new TypedWorker<RQ, RS>(send);
    this.handlers.set(type, t);
    return t;
  }
}

const typedSend =
  (type: string, send: SendFunc): SendFunc =>
  (payload: unknown, transfer?: Transferable[]) =>
    send({ type, payload }, transfer);

export class TypedWorker<RQ, RS = RQ> implements SenderHandler<RQ, RS> {
  private readonly _send: SendFunc;
  handler: HandlerFunc | null;

  constructor(send: SendFunc) {
    this._send = send;
    this.handler = null;
  }

  send(payload: RQ, transfer: Transferable[] = []): void {
    this._send(payload, transfer);
  }

  handle(handler: (payload: RS) => void): void {
    this.handler = handler as HandlerFunc;
  }
}

export const createMockWorkers = (): [RoutedWorker, RoutedWorker] => {
  // eslint-disable-next-line
  let a: RoutedWorker, b: RoutedWorker;
  const aSend = (value: TypedWorkerMessage): void => {
    b.handle({ data: value });
  };
  const bSend = (value: TypedWorkerMessage): void => {
    a.handle({ data: value });
  };
  a = new RoutedWorker(aSend as SendFunc);
  b = new RoutedWorker(bSend as SendFunc);
  return [a, b];
};
