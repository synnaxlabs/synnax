// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Destructor, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

import { AetherLeaf } from "@/core/aether/worker";
import {
  BooleanTelemSink,
  booleanTelemSinkSpec,
  BooleanTelemSource,
  booleanTelemSourceSpec,
  TelemContext,
} from "@/core/vis/telem";
import { AetherNoopTelem } from "@/telem/noop/aether";

export const valveState = z.object({
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

export class AetherValve extends AetherLeaf<typeof valveState, InternalState> {
  static readonly TYPE = "Valve";

  static readonly stateZ = valveState;
  schema = AetherValve.stateZ;
  lastTrigger: number = 0;

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
      this.lastTrigger = performance.now();
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
          console.log(
            TimeSpan.milliseconds(performance.now() - this.lastTrigger).toString()
          );
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
