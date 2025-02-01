// Copyright 2024 Synnax Labs, Inc.
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

export const lightStateZ = z.object({
  enabled: z.boolean(),
  source: telem.booleanSourceSpecZ.optional().default(telem.noopBooleanSourceSpec),
});

export type LightState = z.input<typeof lightStateZ>;

interface InternalState {
  source: telem.BooleanSource;
  addStatus: status.Aggregate;
  stopListening: Destructor;
}

// Light is a component that listens to a telemetry source to update its state.
export class Light
  extends aether.Leaf<typeof lightStateZ, InternalState>
  implements diagram.Element
{
  static readonly TYPE = "Light";

  schema = lightStateZ;

  async afterUpdate(ctx: aether.Context): Promise<void> {
    this.internal.addStatus = status.useOptionalAggregate(ctx);
    const { source: sourceProps } = this.state;
    const { internal: i } = this;
    this.internal.source = await telem.useSource(
      ctx,
      sourceProps,
      this.internal.source,
    );

    await this.updateEnabledState();
    i.stopListening?.();
    i.stopListening = i.source.onChange(() =>
      this.updateEnabledState().catch(this.reportError.bind(this)),
    );
  }

  private reportError(e: Error): void {
    this.internal.addStatus({
      key: this.key,
      variant: "error",
      message: `Failed to update Light: ${e.message}`,
    });
  }

  private async updateEnabledState(): Promise<void> {
    const nextEnabled = await this.internal.source.value();
    if (nextEnabled !== this.state.enabled)
      this.setState((p) => ({ ...p, enabled: nextEnabled }));
  }

  async afterDelete(): Promise<void> {
    await this.internalAfterDelete();
  }

  private async internalAfterDelete(): Promise<void> {
    this.internal.stopListening();
    await this.internal.source.cleanup?.();
  }

  async render(): Promise<void> {}
}

export const REGISTRY: aether.ComponentRegistry = { [Light.TYPE]: Light };
