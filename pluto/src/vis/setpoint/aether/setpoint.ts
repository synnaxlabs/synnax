// Copyright 2025 Synnax Labs, Inc.
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
  trigger: z.number(),
  command: z.number().optional(),
  value: z.number(),
  sink: telem.numberSinkSpecZ.default(telem.noopNumericSinkSpec),
  source: telem.numberSourceSpecZ.default(telem.noopNumericSourceSpec),
});

export type SetpointState = z.input<typeof stateZ>;

interface InternalState {
  source: telem.NumberSource;
  sink: telem.NumberSink;
  stopListening: destructor.Destructor;
  prevTrigger: number;
}

// Setpoint is a component that acts as a switch, commanding a boolean telemetry sink to
// change its value when clicked. It also listens to a boolean telemetry source to
// update its setpoint state.
export class Setpoint
  extends aether.Leaf<typeof stateZ, InternalState>
  implements diagram.Element
{
  static readonly TYPE = "Setpoint";

  schema = stateZ;

  afterUpdate(ctx: aether.Context): void {
    const { sink: sinkProps, source: sourceProps, trigger, command } = this.state;
    const { internal: i } = this;
    i.prevTrigger ??= trigger;
    this.internal.source = telem.useSource(ctx, sourceProps, this.internal.source);
    i.sink = telem.useSink(ctx, sinkProps, i.sink);

    const prevTrigger = i.prevTrigger;
    i.prevTrigger = trigger;

    if (trigger > prevTrigger && command != null) this.internal.sink.set(command);

    this.updateValue();
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => this.updateValue());
  }

  private updateValue(): void {
    const nextValue = this.internal.source.value();
    if (nextValue === this.state.value || isNaN(nextValue)) return;
    this.setState((p) => ({ ...p, value: nextValue, triggered: false }));
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.stopListening?.();
    i.source.cleanup?.();
    i.sink.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Setpoint.TYPE]: Setpoint };
