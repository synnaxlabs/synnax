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

export const toggleStateZ = z.object({
  triggered: z.boolean(),
  enabled: z.boolean(),
  sink: telem.booleanSinkSpecZ.optional().default(telem.noopBooleanSinkSpec),
  source: telem.booleanSourceSpecZ.optional().default(telem.noopBooleanSourceSpec),
});

export type ToggleState = z.input<typeof toggleStateZ>;

interface InternalState {
  source: telem.BooleanSource;
  sink: telem.BooleanSink;
  addStatus: status.Adder;
  stopListening: Destructor;
}

// Toggle is a component that acts as a switch, commanding a boolean telemetry sink to
// change its value when clicked. It also listens to a boolean telemetry source to update
// its toggled state.
export class Toggle
  extends aether.Leaf<typeof toggleStateZ, InternalState>
  implements diagram.Element
{
  static readonly TYPE = "Toggle";

  schema = toggleStateZ;

  afterUpdate(ctx: aether.Context): void {
    this.internal.addStatus = status.useOptionalAdder(ctx);
    const { sink: sinkProps, source: sourceProps, triggered, enabled } = this.state;
    const { triggered: prevTriggered } = this.prevState;
    const { internal: i } = this;
    i.source = telem.useSource(ctx, sourceProps, i.source);
    i.sink = telem.useSink(ctx, sinkProps, i.sink);

    if (triggered && !prevTriggered) i.sink.set(!enabled);
    this.updateEnabledState();
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => this.updateEnabledState());
  }

  private updateEnabledState(): void {
    const nextEnabled = this.internal.source.value();
    if (nextEnabled !== this.state.enabled)
      this.setState((p) => ({ ...p, enabled: nextEnabled, triggered: false }));
  }

  afterDelete(): void {
    this.internal.stopListening?.();
    this.internal.source.cleanup?.();
    this.internal.sink.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Toggle.TYPE]: Toggle };
