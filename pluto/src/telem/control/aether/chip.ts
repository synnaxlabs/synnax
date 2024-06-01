// Copyright 2024 Synnax Labs, Inc.
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

  async afterUpdate(): Promise<void> {
    const { sink: sinkProps, source: sourceProps } = this.state;
    this.internal.source = await telem.useSource(
      this.ctx,
      sourceProps,
      this.internal.source,
    );
    this.internal.sink = await telem.useSink(this.ctx, sinkProps, this.internal.sink);

    if (this.state.triggered && !this.prevState.triggered) {
      this.internal.sink
        .set(this.state.status.data?.authority !== Authority.Absolute.valueOf())
        .catch(console.error);
    }

    await this.updateEnabledState();
    this.internal.stopListening?.();
    this.internal.stopListening = this.internal.source.onChange(() => {
      void this.updateEnabledState();
    });
  }

  private async updateEnabledState(): Promise<void> {
    const nextStatus = await this.internal.source.value();
    if (!nextStatus.time.equals(this.state.status.time)) {
      this.setState((p) => ({ ...p, status: nextStatus, triggered: false }));
    }
  }

  async afterDelete(): Promise<void> {
    this.asyncAfterDelete().catch(console.error);
  }

  private async asyncAfterDelete(): Promise<void> {
    this.internal.stopListening();
    await this.internal.source.cleanup?.();
    await this.internal.sink.cleanup?.();
  }

  render(): void {}
}
