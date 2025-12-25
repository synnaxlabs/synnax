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
import { status } from "@/status/aether";
import { telem } from "@/telem/aether";
import { type diagram } from "@/vis/diagram/aether";

export const toggleStateZ = z.object({
  enabled: z.boolean(),
  sink: telem.booleanSinkSpecZ.default(telem.noopBooleanSinkSpec),
  source: telem.booleanSourceSpecZ.default(telem.noopBooleanSourceSpec),
});

export type ToggleState = z.input<typeof toggleStateZ>;

/** Methods schema for Toggle RPC */
export const toggleMethodsZ = {
  toggle: z.function({ input: z.tuple([]), output: z.void() }),
};

interface InternalState {
  source: telem.BooleanSource;
  sink: telem.BooleanSink;
  addStatus: status.Adder;
  stopListening: destructor.Destructor;
}

// Toggle is a component that acts as a switch, commanding a boolean telemetry sink to
// change its value when clicked. It also listens to a boolean telemetry source to update
// its toggled state.
export class Toggle
  extends aether.Leaf<typeof toggleStateZ, InternalState, typeof toggleMethodsZ>
  implements diagram.Element, aether.HandlersFromSchema<typeof toggleMethodsZ>
{
  static readonly TYPE = "Toggle";
  static readonly METHODS = toggleMethodsZ;

  schema = toggleStateZ;
  methods = toggleMethodsZ;

  afterUpdate(ctx: aether.Context): void {
    this.internal.addStatus = status.useOptionalAdder(ctx);
    const { sink: sinkProps, source: sourceProps } = this.state;
    const { internal: i } = this;
    i.source = telem.useSource(ctx, sourceProps, i.source);
    i.sink = telem.useSink(ctx, sinkProps, i.sink);

    this.updateEnabledState();
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => this.updateEnabledState());
  }

  toggle(): void {
    const { enabled } = this.state;
    this.internal.sink.set(!enabled);
  }

  private updateEnabledState(): void {
    const nextEnabled = this.internal.source.value();
    if (nextEnabled !== this.state.enabled)
      this.setState((p) => ({ ...p, enabled: nextEnabled }));
  }

  afterDelete(): void {
    this.internal.stopListening?.();
    this.internal.source.cleanup?.();
    this.internal.sink.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Toggle.TYPE]: Toggle };
