// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { control, type destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { StateProvider, sugaredStateZ } from "@/telem/control/aether/state";

export const legendStateZ = z.object({
  needsControlOf: channel.keyZ.array(),
  states: sugaredStateZ.array(),
});

interface InternalState {
  stateProv: StateProvider;
  disconnectStateProv?: destructor.Destructor;
}

export class Legend extends aether.Leaf<typeof legendStateZ, InternalState> {
  static readonly TYPE = "Legend";
  schema = legendStateZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.stateProv = StateProvider.use(ctx);

    const keys = this.state.needsControlOf;
    i.disconnectStateProv?.();
    const filter = control.filterTransfersByChannelKey(...keys);
    const states = i.stateProv.get(keys);
    this.setState((p) => ({ ...p, states }));
    i.disconnectStateProv = i.stateProv.onChange((t) => {
      if (filter(t).length === 0) return;
      const states = i.stateProv.get(keys);
      this.setState((p) => ({ ...p, states }));
    });
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.disconnectStateProv?.();
  }

  render(): void {}
}
