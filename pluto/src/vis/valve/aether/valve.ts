// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { AetherNoopTelem } from "@/telem/noop/aether";
import {
  BooleanTelemSink,
  booleanTelemSinkSpec,
  BooleanTelemSource,
  booleanTelemSourceSpec,
  TelemContext,
} from "@/vis/telem";

export const valveStateZ = z.object({
  triggered: z.boolean(),
  active: z.boolean(),
  sink: booleanTelemSinkSpec.optional().default(AetherNoopTelem.booleanSinkSpec),
  source: booleanTelemSourceSpec.optional().default(AetherNoopTelem.booleanSourceSpec),
});

interface InternalState {
  source: BooleanTelemSource;
  cleanupSource: Destructor;
  sink: BooleanTelemSink;
  cleanupSink: Destructor;
}

export class Valve extends aether.Leaf<typeof valveStateZ, InternalState> {
  static readonly TYPE = "Valve";

  schema = valveStateZ;

  afterUpdate(): void {
    const [source, cleanupSource] = TelemContext.use<BooleanTelemSource>(
      this.ctx,
      `${this.key}-source`,
      this.state.source
    );
    const [sink, cleanupSink] = TelemContext.use<BooleanTelemSink>(
      this.ctx,
      `${this.key}-sink`,
      this.state.sink
    );

    this.internal.source = source;
    this.internal.cleanupSource = cleanupSource;
    this.internal.sink = sink;
    this.internal.cleanupSink = cleanupSink;

    if (this.state.triggered !== this.prevState.triggered) {
      this.internal.sink.set(!this.state.active).catch(console.error);
    }

    this.internal.source
      .value()
      .then(() => {
        this.internal.source.onChange(() => {
          this.internal.source
            .value()
            .then((v) => this.setState((p) => ({ ...p, active: v, triggered: false })))
            .catch(console.error);
        });
      })
      .catch(console.error);
  }

  afterDelete(): void {
    this.internal.cleanupSink();
    this.internal.cleanupSource();
  }

  render(): void {}
}
