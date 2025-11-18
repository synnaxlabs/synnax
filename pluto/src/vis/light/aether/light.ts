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
import { telem } from "@/telem/aether";
import { type diagram } from "@/vis/diagram/aether";

export const stateZ = z.object({
  enabled: z.boolean(),
  source: telem.booleanSourceSpecZ.default(telem.noopBooleanSourceSpec),
});
export interface State extends z.input<typeof stateZ> {}

interface InternalState {
  source: telem.BooleanSource;
  stopListening: Destructor;
}

// Light is a component that listens to a telemetry source to update its state.
export class Light
  extends aether.Leaf<typeof stateZ, InternalState>
  implements diagram.Element
{
  static readonly TYPE = "Light";

  schema = stateZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    this.internal.source = telem.useSource(ctx, this.state.source, i.source);
    this.updateEnabledState();
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => this.updateEnabledState());
  }

  private updateEnabledState(): void {
    const nextEnabled = this.internal.source.value();
    if (nextEnabled !== this.state.enabled)
      this.setState((p) => ({ ...p, enabled: nextEnabled }));
  }

  afterDelete(): void {
    this.internal.stopListening?.();
    this.internal.source.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Light.TYPE]: Light };
