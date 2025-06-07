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

export const lightStateZ = z.object({
  enabled: z.boolean(),
  source: telem.booleanSourceSpecZ.optional().default(telem.noopBooleanSourceSpec),
});

export type LightState = z.input<typeof lightStateZ>;

interface InternalState {
  source: telem.BooleanSource;
  addStatus: status.Adder;
  stopListening: Destructor;
}

// Light is a component that listens to a telemetry source to update its state.
export class Light
  extends aether.Leaf<typeof lightStateZ, InternalState>
  implements diagram.Element
{
  static readonly TYPE = "Light";

  schema = lightStateZ;

  afterUpdate(ctx: aether.Context): void {
    this.internal.addStatus = status.useOptionalAdder(ctx);
    const { source: sourceProps } = this.state;
    const { internal: i } = this;
    this.internal.source = telem.useSource(ctx, sourceProps, this.internal.source);
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => this.updateEnabledState());
  }

  private updateEnabledState(): void {
    const nextEnabled = this.internal.source.value();
    if (nextEnabled !== this.state.enabled)
      this.setState((p) => ({ ...p, enabled: nextEnabled }));
  }

  afterDelete(): void {
    this.internal.stopListening();
    this.internal.source.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Light.TYPE]: Light };
