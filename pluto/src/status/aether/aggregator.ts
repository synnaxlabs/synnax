// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Optional, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { Spec, specZ } from "@/status/aether/types";

export const aggregatorStateZ = z.object({
  statuses: specZ.array(),
});

const CONTEXT_KEY = "status.aggregator";

export type Aggreagate = (spec: Optional<Spec, "time">) => void;

export class Aggregator extends aether.Composite<typeof aggregatorStateZ> {
  static readonly TYPE: string = "status.Aggregator";
  schema = aggregatorStateZ;

  afterUpdate(): void {
    this.ctx.set(CONTEXT_KEY, this);
  }

  add(spec: Optional<Spec, "time">): void {
    this.setState((p) => ({
      ...p,
      statuses: [...p.statuses, { time: TimeStamp.now(), ...spec }],
    }));
  }
}

export const useAggregator = (ctx: aether.Context): Aggreagate => {
  const agg = ctx.get<Aggregator>(CONTEXT_KEY);
  return agg.add.bind(agg);
};

export const REGISTRY: aether.ComponentRegistry = {
  [Aggregator.TYPE]: Aggregator,
};
