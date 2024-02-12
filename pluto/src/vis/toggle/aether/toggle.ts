// Copyright 2023 Synnax Labs, Inc.
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
import { telem } from "@/telem/aether";

export const toggleStateZ = z.object({
  triggered: z.boolean(),
  enabled: z.boolean(),
  sink: telem.booleanSinkSpecZ.optional().default(telem.noopBooleanSinkSpec),
  source: telem.booleanSourceSpecZ.optional().default(telem.noopBooleanSourceSpec),
});

interface InternalState {
  source: telem.BooleanSource;
  sink: telem.BooleanSink;
  stopListening: Destructor;
}

export class Toggle extends aether.Leaf<typeof toggleStateZ, InternalState> {
  static readonly TYPE = "Toggle";

  schema = toggleStateZ;

  afterUpdate(): void {
    this.internalAfterUpdate().catch(console.error);
  }

  private async internalAfterUpdate(): Promise<void> {
    const { sink: sinkProps, source: sourceProps } = this.state;
    this.internal.source = await telem.useSource(
      this.ctx,
      sourceProps,
      this.internal.source,
    );
    this.internal.sink = await telem.useSink(this.ctx, sinkProps, this.internal.sink);

    if (this.state.triggered && !this.prevState.triggered) {
      this.internal.sink.set(!this.state.enabled).catch(console.error);
    }

    await this.updateEnabledState();
    this.internal.stopListening?.();
    this.internal.stopListening = this.internal.source.onChange(() => {
      void this.updateEnabledState();
    });
  }

  private async updateEnabledState(): Promise<void> {
    const nextEnabled = await this.internal.source.value();
    if (nextEnabled !== this.state.enabled)
      this.setState((p) => ({ ...p, enabled: nextEnabled, triggered: false }));
  }

  afterDelete(): void {
    this.internalAfterDelete().catch(console.error);
  }

  private async internalAfterDelete(): Promise<void> {
    this.internal.stopListening();
    await this.internal.source.cleanup?.();
    await this.internal.sink.cleanup?.();
  }

  render(): void {}
}

export const REGISTRY: aether.ComponentRegistry = {
  [Toggle.TYPE]: Toggle,
};
