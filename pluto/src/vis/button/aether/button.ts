// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { zod } from "@synnaxlabs/x";
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

/** Methods schema for Button RPC */
export const buttonMethodsZ = {
  onMouseDown: zod.callable(),
  onMouseUp: zod.callable(),
};

interface InternalState {
  sink: telem.BooleanSink;
}

export class Button extends aether.Leaf<typeof buttonStateZ, InternalState> {
  static readonly TYPE = "Button";
  static readonly METHODS = buttonMethodsZ;

  schema = buttonStateZ;

  constructor(props: aether.ComponentConstructorProps) {
    super(props);
    this.bindMethods(buttonMethodsZ, {
      onMouseDown: this.handleMouseDown.bind(this),
      onMouseUp: this.handleMouseUp.bind(this),
    });
  }

  afterUpdate(ctx: aether.Context): void {
    const { sink: sinkProps } = this.state;
    this.internal.sink = telem.useSink(ctx, sinkProps, this.internal.sink);
  }

  private handleMouseDown(): void {
    const { mode } = this.state;
    if (mode === "momentary") this.internal.sink.set(true);
    else if (mode === "pulse") this.internal.sink.set(true, false);
  }

  private handleMouseUp(): void {
    const { mode } = this.state;
    if (mode === "fire") this.internal.sink.set(true);
    else if (mode === "momentary") this.internal.sink.set(false);
  }

  afterDelete(): void {
    this.internal.sink.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Button.TYPE]: Button };
