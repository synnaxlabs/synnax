// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Authority } from "@synnaxlabs/client";
import { type Destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { status } from "@/status/aether";
import { telem } from "@/telem/aether";

export const chipStateZ = z.object({
  triggered: z.boolean(),
  status: status.specZ,
  sink: telem.booleanSinkSpecZ.optional().default(telem.noopBooleanSinkSpec),
  source: telem.statusSourceSpecZ.optional().default(telem.noopStatusSourceSpec),
});

interface InternalState {
  source: telem.StatusSource;
  sink: telem.BooleanSink;
  stopListening: Destructor;
}

export class Chip extends aether.Leaf<typeof chipStateZ, InternalState> {
  static readonly TYPE = "Chip";

  schema = chipStateZ;

  afterUpdate(): void {
    const { sink: sinkProps, source: sourceProps } = this.state;
    this.internal.source = telem.useSource(this.ctx, sourceProps, this.internal.source);
    this.internal.sink = telem.useSink(this.ctx, sinkProps, this.internal.sink);

    if (this.state.triggered && !this.prevState.triggered) {
      this.internal.sink
        .set(this.state.status.data.authority !== Authority.ABSOLUTE.valueOf())
        .catch(console.error);
    }

    void (async () => {
      await this.updateEnabledState();
      this.internal.stopListening?.();
      this.internal.stopListening = this.internal.source.onChange(() => {
        void this.updateEnabledState();
      });
    })();
  }

  private async updateEnabledState(): Promise<void> {
    const nextStatus = await this.internal.source.value();
    if (!nextStatus.time.equals(this.state.status.time))
      this.setState((p) => ({ ...p, status: nextStatus, triggered: false }));
  }

  afterDelete(): void {
    this.internal.source.cleanup?.();
    this.internal.sink.cleanup?.();
  }

  render(): void {}
}
