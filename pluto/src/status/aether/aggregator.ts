// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { type CrudeSpec, specZ } from "@/status/aether/types";

export const aggregatorStateZ = z.object({ statuses: specZ.array() });
export interface AggregatorState extends z.infer<typeof aggregatorStateZ> {}

const CONTEXT_KEY = "status.aggregator";

export class Aggregator extends aether.Composite<typeof aggregatorStateZ> {
  static readonly TYPE: string = "status.Aggregator";
  schema = aggregatorStateZ;

  async afterUpdate(ctx: aether.Context): Promise<void> {
    ctx.set(CONTEXT_KEY, this);
  }

  add(spec: CrudeSpec): void {
    this.setState((p) => ({
      ...p,
      statuses: [...p.statuses, { time: TimeStamp.now(), ...spec, key: id.id() }],
    }));
  }
}

export interface Adder {
  (spec: CrudeSpec): void;
}

export const useAdder = (ctx: aether.Context): Adder => {
  const agg = ctx.get<Aggregator>(CONTEXT_KEY);
  return agg.add.bind(agg);
};

export const useOptionalAdder = (ctx: aether.Context): Adder => {
  const agg = ctx.getOptional<Aggregator>(CONTEXT_KEY);
  if (agg != null) return agg.add.bind(agg);
  return () => {};
};

export interface ExceptionHandler {
  (exc: unknown, message?: string): void;
}

export const useExceptionHandler = (ctx: aether.Context): ExceptionHandler => {
  const addStatus = useAdder(ctx);
  return (exc: unknown, message?: string): void => {
    if (!(exc instanceof Error)) throw exc;
    addStatus({
      variant: "error",
      message: message ?? exc.message,
      description: message != null ? exc.message : undefined,
    });
  };
};

export const REGISTRY: aether.ComponentRegistry = {
  [Aggregator.TYPE]: Aggregator,
};
