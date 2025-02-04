// Copyright 2024 Synnax Labs, Inc.
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

export const buttonStateZ = z.object({
  trigger: z.number(),
  sink: telem.booleanSinkSpecZ.optional().default(telem.noopBooleanSinkSpec),
});

interface InternalState {
  sink: telem.BooleanSink;
  prevTrigger: number;
}

export class Button extends aether.Leaf<typeof buttonStateZ, InternalState> {
  static readonly TYPE = "Button";

  schema = buttonStateZ;

  async afterUpdate(ctx: aether.Context): Promise<void> {
    const { sink: sinkProps } = this.state;
    this.internal.prevTrigger ??= this.state.trigger;
    this.internal.sink = await telem.useSink(ctx, sinkProps, this.internal.sink);
    const prevTrigger = this.internal.prevTrigger;
    this.internal.prevTrigger = this.state.trigger;
    if (this.state.trigger <= prevTrigger) return;
    await this.internal.sink.set(true);
  }

  render(): void {}

  async afterDelete(): Promise<void> {
    await this.internalAfterDelete();
  }

  private async internalAfterDelete(): Promise<void> {
    const { internal: i } = this;
    await i.sink.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Button.TYPE]: Button,
};
