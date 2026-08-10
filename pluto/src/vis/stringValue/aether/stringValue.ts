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

export const stateZ = staleness.stateZ.extend({
  telem: telem.stringSourceSpecZ.default(telem.noopStringSourceSpec),
  // value is written by the worker and read by the DOM, which does the actual text
  // rendering. Arbitrary text needs font fallback and shaping that a monospaced
  // canvas atlas can't provide.
  value: z.string().default(""),
});

export interface State extends z.input<typeof stateZ> {}

interface InternalState {
  source: telem.StringSource;
  stopListening: destructor.Destructor;
  staleness: staleness.Registration;
}

export class StringValue
  extends aether.Leaf<typeof stateZ, InternalState>
  implements diagram.Element
{
  static readonly TYPE = "StringValue";
  static readonly z = stateZ;

  schema = StringValue.z;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.source = telem.useSource(ctx, this.state.telem, i.source);
    i.staleness = staleness.useRegistration(ctx, i.staleness, {
      timeout: () => this.state.stalenessTimeout,
      stale: () => this.state.stale,
      onChange: (stale) => this.setState((p) => ({ ...p, stale })),
    });
    this.publish();
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => {
      i.staleness.received();
      this.publish();
    });
  }

  /** Pushes the source's current value into aether state, where the DOM reads it. */
  private publish(): void {
    const value = this.internal.source.value();
    if (value === this.state.value) return;
    this.setState((p) => ({ ...p, value }));
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.stopListening?.();
    i.staleness.cleanup();
    i.source.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [StringValue.TYPE]: StringValue,
};
