// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection, Synnax, synnaxPropsZ } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";

const stateZ = z.object({
  props: synnaxPropsZ.nullable(),
  state: Synnax.connectivity.connectionStateZ.nullable(),
});

export interface ContextValue {
  synnax: Synnax | null;
  state: connection.State;
}

export const ZERO_CONTEXT_VALUE: ContextValue = {
  synnax: null,
  state: Synnax.connectivity.DEFAULT,
};

export class Provider extends aether.Composite<typeof stateZ, ContextValue> {
  static readonly TYPE = "synnax.Provider";
  static readonly stateZ = stateZ;
  schema = Provider.stateZ;

  async afterUpdate(ctx: aether.Context): Promise<void> {
    if (!ctx.setPreviously(CONTEXT_KEY)) set(ctx, ZERO_CONTEXT_VALUE);
    if (this.state.props == null) {
      if (this.internal.synnax != null) {
        this.setState((p) => ({ ...p, state: Synnax.connectivity.DEFAULT }));
        this.internal.synnax?.close();
        this.internal.synnax = null;
      }
      set(ctx, this.internal);
      return;
    }

    if (
      this.prevState.props != null &&
      deep.equal(this.state.props, this.prevState.props) &&
      this.internal.synnax != null
    )
      return;

    this.internal.synnax = new Synnax(this.state.props);
    this.internal.synnax.connectivity.onChange((state) =>
      this.setState((p) => ({ ...p, state })),
    );
    set(ctx, this.internal);
  }
}

const CONTEXT_KEY = "pluto-client-context";

const set = (ctx: aether.Context, value: ContextValue): void =>
  ctx.set(CONTEXT_KEY, value);

export const use = (ctx: aether.Context): Synnax | null =>
  ctx.get<ContextValue>(CONTEXT_KEY).synnax;

export const REGISTRY: aether.ComponentRegistry = {
  [Provider.TYPE]: Provider,
};
