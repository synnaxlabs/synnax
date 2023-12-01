// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { aether } from "@/aether/aether";
import { status } from "@/status/aether";
import { telem } from "@/telem/aether";
import { type pid } from "@/vis/pid/aether";

export const chipStateZ = z.object({
  source: telem.statusSourceSpecZ.optional().default(telem.noopStatusSourceSpec),
  sink: telem.booleanSinkSpecZ.optional().default(telem.noopBooleanSinkSpec),
  trigger: z.number(),
  status: status.specZ,
});

interface InternalState {
  source: telem.StatusSource;
  sink: telem.BooleanSink;
}

export class Chip
  extends aether.Leaf<typeof chipStateZ, InternalState>
  implements pid.Element
{
  static readonly TYPE = "Chip";
  schema = chipStateZ;

  afterDelete(): void {
    this.internal.source.cleanup?.();
    this.internal.sink.cleanup?.();
  }

  afterUpdate(): void {
    const { internal: i } = this;
    i.source = telem.useSource(this.ctx, this.state.source, i.source);
    i.sink = telem.useSink(this.ctx, this.state.sink, i.sink);

    void i.source.value();

    if (this.state.trigger > this.prevState.trigger)
      i.sink.set(this.state.status.variant !== "success").catch(console.error);

    i.source.onChange(() => {
      this.internal.source
        .value()
        .then((value) => {
          this.setState((p) => ({ ...p, status: value }));
        })
        .catch(console.error);
    });
  }

  async render(): Promise<void> {}
}

export const REGISTRY: aether.ComponentRegistry = {
  [Chip.TYPE]: Chip,
};
