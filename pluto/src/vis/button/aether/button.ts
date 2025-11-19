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
  trigger: z.number(),
  sink: telem.booleanSinkSpecZ.default(telem.noopBooleanSinkSpec),
  mode: modeZ.default("fire"),
});

interface InternalState {
  sink: telem.BooleanSink;
  prevTrigger: number;
}

export const MOUSE_DOWN_INCREMENT = 2;
export const MOUSE_UP_INCREMENT = 1;

export class Button extends aether.Leaf<typeof buttonStateZ, InternalState> {
  static readonly TYPE = "Button";

  schema = buttonStateZ;

  afterUpdate(ctx: aether.Context): void {
    const { sink: sinkProps, mode, trigger } = this.state;
    const { internal: i } = this;
    i.prevTrigger ??= trigger;
    i.sink = telem.useSink(ctx, sinkProps, i.sink);
    const prevTrigger = i.prevTrigger;
    i.prevTrigger = trigger;
    const isMouseDown = trigger === prevTrigger + MOUSE_DOWN_INCREMENT;
    const isMouseUp = trigger === prevTrigger + MOUSE_UP_INCREMENT;
    if (isMouseUp) {
      if (mode == "fire") this.internal.sink.set(true);
      else if (mode == "momentary") this.internal.sink.set(false);
    } else if (isMouseDown)
      if (mode == "momentary") this.internal.sink.set(true);
      else if (mode == "pulse") this.internal.sink.set(true, false);
  }

  afterDelete(): void {
    this.internal.sink.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Button.TYPE]: Button };
