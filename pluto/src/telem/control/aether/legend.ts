// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type control } from "@synnaxlabs/client";
import { color, type destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { flux } from "@/flux/aether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import {
  type ColoredState,
  coloredStateZ,
  Colors,
} from "@/telem/control/aether/colors";
import { listDefinition, type ListQuery } from "@/telem/control/aether/queries";

export const legendStateZ = z.object({
  needsControlOf: channel.keyZ.array(),
  states: coloredStateZ.array(),
  colors: z.record(z.string(), color.colorZ).default({}),
});

interface InternalState {
  colors: Colors;
  retrieve: flux.Retrieve<ListQuery, control.KeyedState[]>;
  disconnectColors?: destructor.Destructor;
}

export class Legend extends aether.Leaf<typeof legendStateZ, InternalState> {
  static readonly TYPE = "Legend";
  schema = legendStateZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.colors = Colors.use(ctx);
    i.colors.setOverrides(this.state.colors);
    const runAsync = status.useErrorHandler(ctx);
    i.retrieve ??= new flux.Retrieve({
      definition: listDefinition,
      onChange: () => this.publish(),
      onError: (error) =>
        runAsync(async () => {
          throw error;
        }, "failed to retrieve control state"),
    });
    i.disconnectColors ??= i.colors.onChange(() => this.publish());
    i.retrieve.update(synnax.use(ctx), { keys: this.state.needsControlOf });
    this.publish();
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.disconnectColors?.();
    i.retrieve.close();
  }

  private publish(): void {
    const { colors, retrieve } = this.internal;
    const states: ColoredState[] = (retrieve.value ?? []).map((s) => ({
      ...s,
      subjectColor: colors.get(s.subject.key),
    }));
    this.setState((p) => ({ ...p, states }));
  }

  render(): void {}
}
