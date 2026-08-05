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
  telem: telem.stringSourceSpecZ.default(telem.noopStringSourceSpec),
  // value is written by the worker and read by the DOM, which does the actual text
  // rendering. Arbitrary text needs font fallback and shaping that a monospaced
  // canvas atlas can't provide.
  value: z.string().default(""),
  // stale reports that no sample has arrived within stalenessTimeout.
  stale: z.boolean().default(false),
  // Seconds without a sample before the value is considered stale.
  stalenessTimeout: z.number().default(5),
});

export interface State extends z.input<typeof stateZ> {}

interface InternalState {
  source: telem.StringSource;
  stopListening: destructor.Destructor;
  staleTimeout?: ReturnType<typeof setTimeout>;
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
    this.publish(false);
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => this.publish(true));
  }

  /**
   * Pushes the source's current value into aether state, where the DOM reads it.
   * A received value clears staleness and restarts the countdown; a value read
   * because props changed leaves staleness alone.
   */
  private publish(received: boolean): void {
    const value = this.internal.source.value();
    if (received) this.restartStaleTimeout();
    const stale = received ? false : this.state.stale;
    if (value === this.state.value && stale === this.state.stale) return;
    this.setState((p) => ({ ...p, value, stale }));
  }

  private restartStaleTimeout(): void {
    const { internal: i } = this;
    clearTimeout(i.staleTimeout);
    i.staleTimeout = setTimeout(
      () => this.setState((p) => ({ ...p, stale: true })),
      this.state.stalenessTimeout * 1000,
    );
  }

  afterDelete(): void {
    const { internal: i } = this;
    clearTimeout(i.staleTimeout);
    i.stopListening?.();
    i.source.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [StringValue.TYPE]: StringValue,
};
