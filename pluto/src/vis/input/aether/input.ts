// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { type diagram } from "@/vis/diagram/aether";

export const stateZ = z.object({
  sink: telem.stringSinkSpecZ.default(telem.noopStringSinkSpec),
  source: telem.stringSourceSpecZ.default(telem.noopStringSourceSpec),
  value: z.string().nullable().default(null),
});

export interface State extends z.input<typeof stateZ> {}

export const methodsZ = {
  set: z.function({ input: z.tuple([z.string()]), output: z.void() }),
};

interface InternalState {
  sink: telem.StringSink;
  source: telem.StringSource;
  stopListening?: destructor.Destructor;
}

export class Input
  extends aether.Leaf<typeof stateZ, InternalState, typeof methodsZ>
  implements diagram.Element, aether.HandlersFromSchema<typeof methodsZ>
{
  static readonly TYPE = "Input";
  static readonly METHODS = methodsZ;
  static readonly z = stateZ;

  schema = stateZ;
  methods = methodsZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.sink = telem.useSink(ctx, this.state.sink, i.sink);
    i.source = telem.useSource(ctx, this.state.source, i.source);
    this.updateValue();
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => this.updateValue());
  }

  // An empty string reads as null: string sources report it before the first sample.
  private updateValue(): void {
    const v = this.internal.source.value().data;
    const next = v === "" ? null : v;
    if (next !== this.state.value) this.setState((p) => ({ ...p, value: next }));
  }

  set(value: string): void {
    this.internal.sink.set(value);
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.stopListening?.();
    i.source.cleanup?.();
    i.sink.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Input.TYPE]: Input };
