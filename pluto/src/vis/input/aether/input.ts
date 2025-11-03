// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { type diagram } from "@/vis/diagram/aether";

export const stateZ = z.object({
  trigger: z.number(),
  command: z.string().optional(),
  sink: telem.stringSinkSpecZ.default(telem.noopStringSinkSpec),
});

export interface State extends z.input<typeof stateZ> {}

interface InternalState {
  sink: telem.StringSink;
  prevTrigger: number;
}

// Input is a component that allows the user to send a string value down a string sink.
export class Input
  extends aether.Leaf<typeof stateZ, InternalState>
  implements diagram.Element
{
  static readonly TYPE = "Input";

  schema = stateZ;

  afterUpdate(ctx: aether.Context): void {
    const { sink: spec, trigger, command } = this.state;
    const { internal: i } = this;
    i.sink = telem.useSink(ctx, spec, i.sink);
    const prevTrigger = i.prevTrigger ?? trigger;
    if (trigger > prevTrigger && command != null) this.internal.sink.set(command);
    i.prevTrigger = trigger;
  }

  afterDelete(): void {
    this.internal.sink.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Input.TYPE]: Input };
