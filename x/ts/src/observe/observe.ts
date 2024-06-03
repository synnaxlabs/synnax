// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Destructor } from "@/destructor";

export type Handler<T> = (value: T) => void;

export interface Observable<T> {
  onChange: (handler: Handler<T>) => Destructor;
}

export interface ObservableAsyncCloseable<T> extends Observable<T> {
  close: () => Promise<void>;
}

export type Transform<I, O> = (value: I) => [O, true] | [O | null, false];

export class Observer<I, O = I> implements Observable<O> {
  private readonly handlers: Map<Handler<O>, null>;
  private readonly transform?: Transform<I, O>;

  constructor(transform?: Transform<I, O>, handlers?: Map<Handler<O>, null>) {
    this.transform = transform;
    this.handlers = handlers ?? new Map();
  }

  onChange(handler: Handler<O>): Destructor {
    this.handlers.set(handler, null);
    return () => this.handlers.delete(handler);
  }

  notify(value: I): void {
    let newValue: O = value as unknown as O;
    if (this.transform != null) {
      const [nv, changed] = this.transform(value);
      if (!changed) return;
      newValue = nv;
    }
    this.handlers.forEach((_, handler) => handler(newValue));
  }
}

export class BaseObserver<V> implements Observable<V> {
  private readonly handlers: Map<Handler<V>, null>;

  constructor(handlers?: Map<Handler<V>, null>) {
    this.handlers = handlers ?? new Map();
  }

  onChange(handler: Handler<V>): Destructor {
    this.handlers.set(handler, null);
    return () => this.handlers.delete(handler);
  }

  notify(value: V): void {
    this.handlers.forEach((_, handler) => handler(value));
  }
}
