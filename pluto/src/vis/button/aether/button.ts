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

export const MODES = ["fire", "momentary", "pulse"] as const;
export const modeZ = z.enum(MODES);

export type Mode = z.infer<typeof modeZ>;

export const buttonStateZ = z.object({
  sink: telem.booleanSinkSpecZ.default(telem.noopBooleanSinkSpec),
  mode: modeZ.default("fire"),
});

export const buttonMethodsZ = {
  onMouseDown: z.function({ input: z.tuple([]), output: z.void() }),
  onMouseUp: z.function({ input: z.tuple([]), output: z.void() }),
};

interface InternalState {
  sink: telem.BooleanSink;
}

export class Button
  extends aether.Leaf<typeof buttonStateZ, InternalState, typeof buttonMethodsZ>
  implements aether.HandlersFromSchema<typeof buttonMethodsZ>
{
  static readonly TYPE = "Button";
  static readonly METHODS = buttonMethodsZ;

  schema = buttonStateZ;
  methods = buttonMethodsZ;

  afterUpdate(ctx: aether.Context): void {
    const { sink: sinkProps } = this.state;
    this.internal.sink = telem.useSink(ctx, sinkProps, this.internal.sink);
  }

  onMouseDown(): void {
    const { mode } = this.state;
    if (mode === "momentary") this.internal.sink.set(true);
    else if (mode === "pulse") this.internal.sink.set(true, false);
  }

  onMouseUp(): void {
    const { mode } = this.state;
    if (mode === "fire") this.internal.sink.set(true);
    else if (mode === "momentary") this.internal.sink.set(false);
  }

  afterDelete(): void {
    this.internal.sink.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Button.TYPE]: Button };
