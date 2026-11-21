// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { destructor } from "@/destructor";
import { withTimeout } from "@/sync/timeout";
import { type CrudeTimeSpan, TimeSpan } from "@/telem";

/** Handler is called when the value of an Observable changes. */
export type Handler<T> = (value: T) => void;

/** A generic interface for an entity whose value can be observed when it changes. */
export interface Observable<T> {
  /**
   * Binds the given handler to the Observable, and starts calling it whenever the value
   * of the Observable changes.
   * @returns A function that can be called to unbind the handler from the Observable.
   */
  onChange: (handler: Handler<T>) => destructor.Destructor;
}

/**
 * Resolves with the first observed value satisfying the predicate. The current value is
 * evaluated first, so an already-satisfied observable resolves without waiting.
 * @param current - Reads the observable's present value, which `onChange` alone cannot
 * supply.
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
  let detach: destructor.Destructor | undefined;
  const satisfied = new Promise<V>((resolve) => {
    detach = observable.onChange((value) => {
      if (predicate(value)) resolve(value);
    });
  });
  try {
    if (timeout == null) return await satisfied;
    const span = new TimeSpan(timeout);
    return await withTimeout(
      satisfied,
      span,
      () => new Error(`timed out after ${span.toString()}`),
    );
  } finally {
    detach?.();
  }
};

/** An Observable that callers notify by hand. */
export class Observer<V> implements Observable<V> {
  private readonly handlers = new Set<Handler<V>>();

  onChange(handler: Handler<V>): destructor.Destructor {
    this.handlers.add(handler);
    return () => this.handlers.delete(handler);
  }

  /** Calls every bound handler with the value. */
  notify(value: V): void {
    this.handlers.forEach((handler) => handler(value));
  }
}
