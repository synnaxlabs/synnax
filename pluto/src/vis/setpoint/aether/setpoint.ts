// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { type diagram } from "@/vis/diagram/aether";

export const stateZ = z.object({
  value: z.number(),
  sink: telem.numberSinkSpecZ.default(telem.noopNumericSinkSpec),
});

export type SetpointState = z.input<typeof stateZ>;

export const methodsZ = {
  set: z.function({ input: z.tuple([z.number()]), output: z.void() }),
};

interface InternalState {
  sink: telem.NumberSink;
}

// Setpoint is a component that commands a numeric telemetry sink to change its value
// when triggered.
export class Setpoint
  extends aether.Leaf<typeof stateZ, InternalState, typeof methodsZ>
  implements diagram.Element, aether.HandlersFromSchema<typeof methodsZ>
{
  static readonly TYPE = "Setpoint";
  static readonly METHODS = methodsZ;

  schema = stateZ;
  methods = methodsZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.sink = telem.useSink(ctx, this.state.sink, i.sink);
  }

  set(value: number): void {
    this.internal.sink.set(value);
  }

  afterDelete(): void {
    this.internal.sink.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Setpoint.TYPE]: Setpoint };
