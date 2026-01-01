// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { type diagram } from "@/vis/diagram/aether";

export const stateZ = z.object({
  value: z.number(),
  sink: telem.numberSinkSpecZ.default(telem.noopNumericSinkSpec),
  source: telem.numberSourceSpecZ.default(telem.noopNumericSourceSpec),
});

export type SetpointState = z.input<typeof stateZ>;

export const methodsZ = {
  set: z.function({ input: z.tuple([z.number()]), output: z.void() }),
};

interface InternalState {
  source: telem.NumberSource;
  sink: telem.NumberSink;
  stopListening: destructor.Destructor;
}

// Setpoint is a component that acts as a switch, commanding a numeric telemetry sink to
// change its value when triggered. It also listens to a numeric telemetry source to
// update its setpoint state.
export class Setpoint
  extends aether.Leaf<typeof stateZ, InternalState, typeof methodsZ>
  implements diagram.Element, aether.HandlersFromSchema<typeof methodsZ>
{
  static readonly TYPE = "Setpoint";
  static readonly METHODS = methodsZ;

  schema = stateZ;
  methods = methodsZ;

  afterUpdate(ctx: aether.Context): void {
    const { sink: sinkProps, source: sourceProps } = this.state;
    const { internal: i } = this;
    i.source = telem.useSource(ctx, sourceProps, i.source);
    i.sink = telem.useSink(ctx, sinkProps, i.sink);
    this.updateValue();
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => this.updateValue());
  }

  set(value: number): void {
    this.internal.sink.set(value);
  }

  private updateValue(): void {
    const nextValue = this.internal.source.value();
    if (nextValue === this.state.value || isNaN(nextValue)) return;
    this.setState((p) => ({ ...p, value: nextValue }));
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.stopListening?.();
    i.source.cleanup?.();
    i.sink.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Setpoint.TYPE]: Setpoint };
