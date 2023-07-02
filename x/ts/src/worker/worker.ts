// Copyright 2023 Synnax Labs, Inc.
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
  payload: any;
}

type SendFunc = (value: any, transfer?: Transferable[]) => void;
type HandlerFunc = (value: any) => void;

export class RoutedWorker {
  sender: SendFunc;
  handlers: Map<string, TypedWorker<any>>;

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
  (payload: any, transfer?: Transferable[]) => {
    return send({ type, payload }, transfer);
  };

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
    this.handler = handler;
  }
}

export const createMockWorkers = (): [RoutedWorker, RoutedWorker] => {
  let a: RoutedWorker, b: RoutedWorker;
  const aSend = (value: any, transfer?: Transferable[]): void => {
    b.handle({ data: value });
  };
  const bSend = (value: any, transfer?: Transferable[]): void => {
    a.handle({ data: value });
  };
  a = new RoutedWorker(aSend);
  b = new RoutedWorker(bSend);
  return [a, b];
};
