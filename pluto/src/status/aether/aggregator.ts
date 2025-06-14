// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, status, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { aether } from "@/aether/aether";

export const aggregatorStateZ = z.object({ statuses: status.statusZ.array() });
export interface AggregatorState extends z.infer<typeof aggregatorStateZ> {}

const CONTEXT_KEY = "status.aggregator";

interface ContextValue {
  add: Adder;
  parse: (spec: status.New) => status.Status;
}

export class Aggregator extends aether.Composite<typeof aggregatorStateZ> {
  static readonly TYPE: string = "status.Aggregator";
  schema = aggregatorStateZ;

  afterUpdate(ctx: aether.Context): void {
    if (ctx.wasSetPreviously(CONTEXT_KEY)) return;
    ctx.set(CONTEXT_KEY, { add: this.add.bind(this), parse: this.parse.bind(this) });
  }

  private parse(spec: status.New): status.Status {
    return { time: TimeStamp.now(), key: id.create(), ...spec };
  }

  private add(spec: status.New): void {
    this.setState((p) => ({
      ...p,
      statuses: [...p.statuses, this.parse(spec)],
    }));
  }
}

export interface Adder {
  (spec: status.New): void;
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

export const fromException = (exc: unknown, message?: string): status.New => {
  if (!(exc instanceof Error)) throw exc;
  return {
    variant: "error",
    message: message ?? exc.message,
    description: message != null ? exc.message : undefined,
  };
};

export const createErrorHandler =
  (add: Adder): ErrorHandler =>
  (excOrFunc: unknown | (() => Promise<void>), message?: string): void => {
    if (typeof excOrFunc !== "function") return add(fromException(excOrFunc, message));
    void (async () => {
      try {
        await excOrFunc();
      } catch (exc) {
        add(fromException(exc, message));
      }
    })();
  };

export const createAsyncExceptionHandler =
  (add: Adder): AsyncErrorHandler =>
  async (func: () => Promise<void>, message?: string): Promise<void> => {
    try {
      await func();
    } catch (exc) {
      add(fromException(exc, message));
    }
  };

export const useErrorHandler = (ctx: aether.Context): ErrorHandler =>
  createErrorHandler(useAdder(ctx));

export const useAsyncErrorHandler = (ctx: aether.Context): AsyncErrorHandler =>
  createAsyncExceptionHandler(useAdder(ctx));

export const REGISTRY: aether.ComponentRegistry = {
  [Aggregator.TYPE]: Aggregator,
};
