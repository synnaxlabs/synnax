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
import { staleness } from "@/vis/staleness/aether";

const stateMappingZ = z.object({
  key: z.string(),
  value: z.number(),
});

export const stateZ = staleness.stateZ.extend({
  key: z.string().nullable().default(null),
  options: z.array(stateMappingZ).default([]),
  source: telem.numberSourceSpecZ.default(telem.noopNumericSourceSpec),
});

export interface State extends z.input<typeof stateZ> {}

interface InternalState {
  source: telem.NumberSource;
  stopListening: destructor.Destructor;
  staleness: staleness.Registration;
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
    i.staleness = staleness.useRegistration(ctx, i.staleness, {
      timeout: () => this.state.stalenessTimeout,
      stale: () => this.state.stale,
      onChange: (stale) => this.setState((p) => ({ ...p, stale })),
    });
    this.updateMatchedOption();
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => {
      i.staleness.received();
      this.updateMatchedOption();
    });
  }

  private updateMatchedOption(): void {
    const nextValue = this.internal.source.value();
    if (isNaN(nextValue)) return;
    const matched = this.state.options.find((o) => o.value === nextValue);
    const nextKey = matched?.key ?? null;
    if (nextKey === this.state.key) return;
    this.setState((p) => ({ ...p, key: nextKey }));
  }

  afterDelete(): void {
    this.internal.stopListening?.();
    this.internal.staleness?.cleanup();
    this.internal.source.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [StateIndicator.TYPE]: StateIndicator,
};
