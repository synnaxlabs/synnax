// Copyright 2026 Synnax Labs, Inc.
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
  sink: telem.stringSinkSpecZ.default(telem.noopStringSinkSpec),
});

export interface State extends z.input<typeof stateZ> {}

export const methodsZ = {
  set: z.function({ input: z.tuple([z.string()]), output: z.void() }),
};

interface InternalState {
  sink: telem.StringSink;
}

// Input is a component that allows the user to send a string value down a string sink.
export class Input
  extends aether.Leaf<typeof stateZ, InternalState, typeof methodsZ>
  implements diagram.Element, aether.HandlersFromSchema<typeof methodsZ>
{
  static readonly TYPE = "Input";
  static readonly METHODS = methodsZ;

  schema = stateZ;
  methods = methodsZ;

  afterUpdate(ctx: aether.Context): void {
    const { sink: spec } = this.state;
    this.internal.sink = telem.useSink(ctx, spec, this.internal.sink);
  }

  set(value: string): void {
    this.internal.sink.set(value);
  }

  afterDelete(): void {
    this.internal.sink.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Input.TYPE]: Input };
