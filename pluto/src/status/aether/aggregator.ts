// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";

export const aggregatorStateZ = z.object({ statuses: status.statusZ().array() });
export interface AggregatorState extends z.infer<typeof aggregatorStateZ> {}

const CONTEXT_KEY = "status.aggregator";

interface ContextValue {
  add: Adder;
  create: (spec: status.Crude) => status.Status;
}

export class Aggregator extends aether.Composite<typeof aggregatorStateZ> {
  static readonly TYPE: string = "status.Aggregator";
  schema = aggregatorStateZ;

  afterUpdate(ctx: aether.Context): void {
    if (ctx.wasSetPreviously(CONTEXT_KEY)) return;
    ctx.set(CONTEXT_KEY, {
      add: this.add.bind(this),
      create: status.create,
    });
  }

  private add(spec: status.Crude): void {
    this.setState((p) => ({
      ...p,
      statuses: [...p.statuses, status.create(spec)],
    }));
  }
}

export interface Adder {
  <D = undefined>(spec: status.Crude<D>): void;
}

export const useAdder = (ctx: aether.Context): Adder =>
  ctx.get<ContextValue>(CONTEXT_KEY).add;

export const useOptionalAdder = (ctx: aether.Context): Adder => {
  const agg = ctx.getOptional<ContextValue>(CONTEXT_KEY);
  if (agg != null) return agg.add;
  return () => {};
};

export interface ErrorHandler {
  (exc: unknown, message?: string): void;
  (func: () => Promise<void>, message?: string): void;
}

export interface AsyncErrorHandler {
  (func: () => Promise<void>, message?: string): Promise<void>;
}

export const createErrorHandler =
  (add: Adder): ErrorHandler =>
  (excOrFunc: unknown | (() => Promise<void> | void), message?: string): void => {
    if (typeof excOrFunc !== "function")
      return add(status.fromException(excOrFunc, message));
    void (async () => {
      try {
        const promise = excOrFunc();
        // Skip the added microtask if the function returns void instead of a promise.
        if (promise != null) await promise;
      } catch (exc) {
        add(status.fromException(exc, message));
      }
    })();
  };

export const createAsyncExceptionHandler =
  (add: Adder): AsyncErrorHandler =>
  async (func: () => Promise<void>, message?: string): Promise<void> => {
    try {
      await func();
    } catch (exc) {
      add(status.fromException(exc, message));
    }
  };

export const useErrorHandler = (ctx: aether.Context): ErrorHandler =>
  createErrorHandler(useAdder(ctx));

export const useAsyncErrorHandler = (ctx: aether.Context): AsyncErrorHandler =>
  createAsyncExceptionHandler(useAdder(ctx));

export const REGISTRY: aether.ComponentRegistry = {
  [Aggregator.TYPE]: Aggregator,
};
