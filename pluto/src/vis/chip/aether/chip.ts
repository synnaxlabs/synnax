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
import { status } from "@/status/aether";
import { telem } from "@/telem/core";
import { noop } from "@/telem/noop";
import { type pid } from "@/vis/pid/aether";

export const chipStateZ = z.object({
  source: telem.statusSourceSpecZ.optional().default(noop.statusSourceSpec),
  sink: telem.booleanSinkSpecZ.optional().default(noop.booleanSinkSpec),
  trigger: z.number(),
  status: status.specZ,
});

interface InternalState {
  source: telem.StatusSource;
  cleanupSource: Destructor;
  sink: telem.BooleanSink;
  cleanupSink: Destructor;
}

export class Chip
  extends aether.Leaf<typeof chipStateZ, InternalState>
  implements pid.Element
{
  static readonly TYPE = "Chip";
  schema = chipStateZ;

  afterUpdate(): void {
    const [source, cleanupSource] = telem.use<telem.StatusSource>(
      this.ctx,
      `${this.key}-source`,
      this.state.source,
    );
    const [sink, cleanupSink] = telem.use<telem.BooleanSink>(
      this.ctx,
      `${this.key}-sink`,
      this.state.sink,
    );
    this.internal.source = source;
    this.internal.cleanupSource = cleanupSource;
    this.internal.sink = sink;
    this.internal.cleanupSink = cleanupSink;

    void source.value();

    if (this.state.trigger > this.prevState.trigger)
      sink.set(this.state.status.variant !== "success").catch(console.error);

    source.onChange(() => {
      this.internal.source
        .value()
        .then((value) => {
          this.setState((p) => ({ ...p, status: value }));
        })
        .catch(console.error);
    });
  }

  async render(): Promise<void> {}
}

export const REGISTRY: aether.ComponentRegistry = {
  [Chip.TYPE]: Chip,
};
