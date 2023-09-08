// Copyrght 2023 Synnax Labs, Inc.
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
import { telem } from "@/telem/core";
import { noop } from "@/telem/noop";

export const buttonStateZ = z.object({
  trigger: z.number(),
  sink: telem.booleanSinkSpecZ.optional().default(noop.booleanSinkSpec),
});

interface InternalState {
  sink: telem.BooleanSink;
  cleanupSink: Destructor;
}

export class Button extends aether.Leaf<typeof buttonStateZ, InternalState> {
  static readonly TYPE = "Button";

  schema = buttonStateZ;

  afterUpdate(): void {
    const [sink, cleanupSink] = telem.use<telem.BooleanSink>(
      this.ctx,
      this.key,
      this.state.sink,
    );
    this.internal.sink = sink;
    this.internal.cleanupSink = cleanupSink;

    if (this.state.trigger > this.prevState.trigger)
      this.internal.sink.set(true).catch(console.error);
  }

  render(): void {}

  afterDelete(): void {
    this.internal.cleanupSink();
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Button.TYPE]: Button,
};
