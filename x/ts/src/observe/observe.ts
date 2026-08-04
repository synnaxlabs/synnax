// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@/destructor";
import { type CrudeTimeSpan, TimeSpan } from "@/telem";

/** Handler is called when the value of an Observable changes. */
export type Handler<T> = (value: T) => void;

export type AsyncHandler<T> = (value: T) => Promise<void>;

/** A generic interface for an entity whose value can be observed when it changes. */
export interface Observable<T> {
  /**
   * Binds the given handler to the Observable, and starts calling it whenever the
   * value of the Observable changes.
   * @param handler The handler to bind to the Observable.
   * @returns A function that can be called to unbind the handler from the Observable.
   */
  onChange: (handler: Handler<T>) => destructor.Destructor;
}

/**
 * Resolves with the first observed value satisfying the predicate. The current
 * value is evaluated first, so an already-satisfied observable resolves without
 * waiting.
 * @param current - Reads the observable's present value, which `onChange` alone
 * cannot supply.
 * @param timeout - How long to wait before rejecting. Omit to wait forever.
 * @throws {Error} if the timeout elapses first.
 */
export const until = async <V>(
  observable: Observable<V>,
  current: () => V,
  predicate: (value: V) => boolean,
  timeout?: CrudeTimeSpan,
): Promise<V> => {
  const value = current();
  if (predicate(value)) return value;
  return await new Promise<V>((resolve, reject) => {
    let timer: ReturnType<typeof setTimeout> | undefined;
    const detach = observable.onChange((value) => {
      if (!predicate(value)) return;
      if (timer != null) clearTimeout(timer);
      detach();
      resolve(value);
    });
    if (timeout != null) {
      const span = new TimeSpan(timeout);
      timer = setTimeout(() => {
        detach();
        reject(new Error(`timed out after ${span.toString()}`));
      }, span.milliseconds);
    }
  });
};

/** An Observable that can be closed using an async function. */
export interface ObservableAsyncCloseable<T> extends Observable<T> {
  /** Closes the Observable. */
  close: () => Promise<void>;
}

/** A function that transforms a value of type I into a value of type O.
 * @param value The value to transform.
 * @returns A tuple containing the transformed value and a boolean indicating whether
 * the value has changed (i.e. whether the observable should notify its handlers).
 */
export type Transform<I, O> = (value: I) => [O, true] | [O | null, false];

/**
 * An implementation fo the Observable interface that can be manually notified of changes.
 */
export class Observer<I, O = I> implements ObservableAsyncCloseable<O> {
  private readonly handlers: Map<Handler<O>, null>;
  private readonly transform?: Transform<I, O>;
  private closer?: () => Promise<void>;

  constructor(transform?: Transform<I, O>, handlers?: Map<Handler<O>, null>) {
    this.transform = transform;
    this.handlers = handlers ?? new Map();
  }

  /** Implements the observable interface. */
  onChange(handler: Handler<O>): destructor.Destructor {
    this.handlers.set(handler, null);
    return () => this.handlers.delete(handler);
  }

  /** Notifies all handlers that the value of the observable has changed. */
  notify(value: I): void {
    let newValue: O = value as unknown as O;
    if (this.transform != null) {
      const [nv, changed] = this.transform(value);
      if (!changed) return;
      newValue = nv;
    }
    this.handlers.forEach((_, handler) => handler(newValue));
  }

  setCloser(closer: () => Promise<void>): void {
    this.closer = closer;
  }

  async close(): Promise<void> {
    return await this.closer?.();
  }
}

export class BaseObserver<V> implements Observable<V> {
  private readonly handlers: Map<Handler<V>, null>;

  constructor(handlers?: Map<Handler<V>, null>) {
    this.handlers = handlers ?? new Map();
  }

  onChange(handler: Handler<V>): destructor.Destructor {
    this.handlers.set(handler, null);
    return () => this.handlers.delete(handler);
  }

  notify(value: V): void {
    this.handlers.forEach((_, handler) => handler(value));
  }
}
