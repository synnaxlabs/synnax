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

  async afterUpdate(ctx: aether.Context): Promise<void> {
    this.internal.addStatus = status.useOptionalAdder(ctx);
    const { sink: sinkProps, source: sourceProps, triggered, enabled } = this.state;
    const { triggered: prevTriggered } = this.prevState;
    const { internal: i } = this;
    i.source = await telem.useSource(ctx, sourceProps, i.source);
    i.sink = await telem.useSink(ctx, sinkProps, i.sink);

    if (triggered && !prevTriggered) await i.sink.set(!enabled);

    await this.updateEnabledState();
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => {
      this.updateEnabledState().catch(this.reportError.bind(this));
    });
  }

  private reportError(e: Error): void {
    this.internal.addStatus({
      key: this.key,
      variant: "error",
      message: `Failed to update Toggle: ${e.message}`,
    });
  }

  private async updateEnabledState(): Promise<void> {
    const nextEnabled = await this.internal.source.value();
    if (nextEnabled !== this.state.enabled)
      this.setState((p) => ({ ...p, enabled: nextEnabled, triggered: false }));
  }

  async afterDelete(): Promise<void> {
    await this.internalAfterDelete();
  }

  private async internalAfterDelete(): Promise<void> {
    this.internal.stopListening();
    await this.internal.source.cleanup?.();
    await this.internal.sink.cleanup?.();
  }

  async render(): Promise<void> {}
}

export const REGISTRY: aether.ComponentRegistry = { [Toggle.TYPE]: Toggle };
