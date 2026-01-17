// Copyright 2026 Synnax Labs, Inc.
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
import {
  type Adder,
  createAsyncErrorHandler,
  createErrorHandler,
  type ErrorHandler,
} from "@/status/aether/errorHandler";
import { type AsyncErrorHandler } from "@/status/base/Aggregator";

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

export const useAdder = (ctx: aether.Context): Adder =>
  ctx.get<ContextValue>(CONTEXT_KEY).add;

export const useOptionalAdder = (ctx: aether.Context): Adder => {
  const agg = ctx.getOptional<ContextValue>(CONTEXT_KEY);
  if (agg != null) return agg.add;
  return () => {};
};
export const useErrorHandler = (ctx: aether.Context): ErrorHandler =>
  createErrorHandler(useAdder(ctx));

export const useAsyncErrorHandler = (ctx: aether.Context): AsyncErrorHandler =>
  createAsyncErrorHandler(useAdder(ctx));

export const REGISTRY: aether.ComponentRegistry = {
  [Aggregator.TYPE]: Aggregator,
};
