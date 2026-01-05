// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control, deep, type destructor, status } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";

export const chipStatusDetailsZ = z
  .object({
    authority: control.authorityZ.optional(),
    valid: z.boolean().optional(),
  })
  .default({ authority: undefined, valid: false });

export const chipStateZ = z.object({
  triggered: z.boolean(),
  status: status.statusZ(chipStatusDetailsZ),
  sink: telem.booleanSinkSpecZ.default(telem.noopBooleanSinkSpec),
  source: telem.statusSourceSpecZ.default(telem.noopStatusSourceSpec),
});

interface InternalState {
  source: telem.StatusSource<typeof chipStatusDetailsZ>;
  sink: telem.BooleanSink;
  stopListening: destructor.Destructor;
}

export class Chip extends aether.Leaf<typeof chipStateZ, InternalState> {
  static readonly TYPE = "Chip";

  schema = chipStateZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    const { sink, source } = this.state;
    i.source = telem.useSource<status.Status<typeof chipStatusDetailsZ>>(
      ctx,
      source,
      i.source,
    );
    i.sink = telem.useSink(ctx, sink, i.sink);
    if (this.state.triggered && !this.prevState.triggered)
      i.sink.set(this.state.status.details?.authority !== control.ABSOLUTE_AUTHORITY);
    this.updateEnabledState();
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => this.updateEnabledState());
  }

  private updateEnabledState(): void {
    const nextStatus = this.internal.source.value();
    if (!deep.equal(nextStatus, this.state.status))
      this.setState((p) => ({ ...p, status: nextStatus, triggered: false }));
  }

  afterDelete(): void {
    this.internal.stopListening();
    this.internal.source.cleanup?.();
    this.internal.sink.cleanup?.();
  }

  render(): void {}
}
