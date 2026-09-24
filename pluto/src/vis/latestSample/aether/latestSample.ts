// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";

export const stateZ = z.object({
  source: telem.numberSourceSpecZ.default(telem.noopNumericSourceSpec),
  time: TimeStamp.z.nullable().default(null),
});

export interface State extends z.input<typeof stateZ> {}

interface InternalState {
  source: telem.NumberSource;
  stopListening: destructor.Destructor;
}

// One second is the display's granularity, so finer pushes only cost renders.
const MIN_STEP = TimeSpan.SECOND.valueOf();

/** Reports the time of the source's newest sample, at most once per second. */
export class LatestSample extends aether.Leaf<typeof stateZ, InternalState> {
  static readonly TYPE = "LatestSample";
  static readonly z = stateZ;

  schema = LatestSample.z;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.source = telem.useSource(ctx, this.state.source, i.source);
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => this.report());
    // value() opens the stream, lastWrite() alone does not.
    i.source.value();
    this.report();
  }

  private report(): void {
    const time = this.internal.source.lastWrite?.() ?? null;
    if (time == null) return;
    const prev = this.state.time;
    if (prev != null && time.valueOf() - prev.valueOf() < MIN_STEP) return;
    this.setState((p) => ({ ...p, time }));
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.stopListening?.();
    i.source.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [LatestSample.TYPE]: LatestSample,
};
