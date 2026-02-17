// Copyright 2026 Synnax Labs, Inc.
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
  value: z.number(),
  source: telem.numberSourceSpecZ.default(telem.noopNumericSourceSpec),
});

export interface State extends z.input<typeof stateZ> {}

interface InternalState {
  source: telem.NumberSource;
  stopListening: destructor.Destructor;
}

export class StateIndicator
  extends aether.Leaf<typeof stateZ, InternalState>
  implements diagram.Element
{
  static readonly TYPE = "StateIndicator";

  schema = stateZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    this.internal.source = telem.useSource(ctx, this.state.source, i.source);
    this.updateValue();
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => this.updateValue());
  }

  private updateValue(): void {
    const nextValue = this.internal.source.value();
    if (nextValue === this.state.value || isNaN(nextValue)) return;
    this.setState((p) => ({ ...p, value: nextValue }));
  }

  afterDelete(): void {
    this.internal.stopListening?.();
    this.internal.source.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [StateIndicator.TYPE]: StateIndicator,
};
