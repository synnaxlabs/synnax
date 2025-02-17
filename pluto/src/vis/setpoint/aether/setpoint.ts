// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { status } from "@/status/aether";
import { telem } from "@/telem/aether";
import { type diagram } from "@/vis/diagram/aether";

export const setpointStateZ = z.object({
  trigger: z.number(),
  command: z.number().optional(),
  value: z.number(),
  sink: telem.numberSinkSpecZ.optional().default(telem.noopNumericSinkSpec),
  source: telem.numberSourceSpecZ.optional().default(telem.noopNumericSourceSpec),
});

export type SetpointState = z.input<typeof setpointStateZ>;

interface InternalState {
  source: telem.NumberSource;
  sink: telem.NumberSink;
  addStatus: status.Adder;
  stopListening: Destructor;
  prevTrigger: number;
}

// Setpoint is a component that acts as a switch, commanding a boolean telemetry sink to
// change its value when clicked. It also listens to a boolean telemetry source to update
// its setpoint state.
export class Setpoint
  extends aether.Leaf<typeof setpointStateZ, InternalState>
  implements diagram.Element
{
  static readonly TYPE = "Setpoint";

  schema = setpointStateZ;

  async afterUpdate(ctx: aether.Context): Promise<void> {
    this.internal.addStatus = status.useOptionalAdder(ctx);
    const { sink: sinkProps, source: sourceProps, trigger, command } = this.state;
    const { internal: i } = this;
    i.prevTrigger ??= trigger;
    this.internal.source = await telem.useSource(
      ctx,
      sourceProps,
      this.internal.source,
    );
    i.sink = await telem.useSink(ctx, sinkProps, i.sink);

    const prevTrigger = i.prevTrigger;
    i.prevTrigger = trigger;
    if (trigger > prevTrigger && command != null) await this.internal.sink.set(command);

    await this.updateValue();
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => {
      this.updateValue().catch(this.reportError.bind(this));
    });
  }

  private reportError(e: Error): void {
    this.internal.addStatus({
      key: this.key,
      variant: "error",
      message: `Failed to update Setpoint: ${e.message}`,
    });
  }

  private async updateValue(): Promise<void> {
    const nextValue = await this.internal.source.value();
    if (nextValue === this.state.value) return;
    this.setState((p) => ({ ...p, value: nextValue, triggered: false }));
  }

  async afterDelete(): Promise<void> {
    const { internal: i } = this;
    i.stopListening();
    await i.source.cleanup?.();
    await i.sink.cleanup?.();
  }

  async render(): Promise<void> {}
}

export const REGISTRY: aether.ComponentRegistry = { [Setpoint.TYPE]: Setpoint };
