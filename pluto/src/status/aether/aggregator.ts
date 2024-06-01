// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { type CrudeSpec,specZ } from "@/status/aether/types";

export const aggregatorStateZ = z.object({
  statuses: specZ.array(),
});

const CONTEXT_KEY = "status.aggregator";

export type Aggregate = (spec: CrudeSpec) => void;

export class Aggregator extends aether.Composite<typeof aggregatorStateZ> {
  static readonly TYPE: string = "status.Aggregator";
  schema = aggregatorStateZ;

  async afterUpdate(): Promise<void> {
    this.ctx.set(CONTEXT_KEY, this);
  }

  add(spec: CrudeSpec): void {
    const time = TimeStamp.now();
    this.setState((p) => ({
      ...p,
      statuses: [...p.statuses, { time, key: time.toString(), ...spec }],
    }));
  }
}

export const useAggregate = (ctx: aether.Context): Aggregate => {
  const agg = ctx.get<Aggregator>(CONTEXT_KEY);
  return agg.add.bind(agg);
};

export const useOptionalAggregate = (ctx: aether.Context): Aggregate => {
  const agg = ctx.getOptional<Aggregator>(CONTEXT_KEY);
  if (agg != null) return agg.add.bind(agg);
  return () => {};
};

export const REGISTRY: aether.ComponentRegistry = {
  [Aggregator.TYPE]: Aggregator,
};
