// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ConnectionState, Synnax, synnaxPropsZ } from "@synnaxlabs/client";
import { Deep } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";

const stateZ = z.object({
  props: synnaxPropsZ.nullable(),
  state: Synnax.connectivity.connectionStateZ.nullable(),
});

export interface ContextValue {
  client: Synnax | null;
  state: ConnectionState;
}

export const ZERO_CONTEXT_VALUE: ContextValue = {
  client: null,
  state: Synnax.connectivity.DEFAULT,
};

export class Provider extends aether.Composite<typeof stateZ, ContextValue> {
  static readonly TYPE = "ClientProvider";
  static readonly stateZ = stateZ;
  schema = Provider.stateZ;

  afterUpdate(): void {
    if (!this.ctx.has(CONTEXT_KEY)) set(this.ctx, ZERO_CONTEXT_VALUE);
    if (this.state.props == null) {
      if (this.internal.client != null) {
        this.setState((p) => ({ ...p, state: Synnax.connectivity.DEFAULT }));
        this.internal.client?.close();
        this.internal.client = null;
      }
      return;
    }

    if (
      this.prevState.props != null &&
      Deep.equal(this.state.props, this.prevState.props) &&
      this.internal.client != null
    ) {
      return;
    }

    this.internal.client = new Synnax(this.state.props);
    this.internal.client.connectivity.onChange((state) =>
      this.setState((p) => ({ ...p, state }))
    );
    set(this.ctx, this.internal);
  }
}

const CONTEXT_KEY = "pluto-client-context";

const set = (ctx: aether.Context, value: ContextValue): void =>
  ctx.set(CONTEXT_KEY, value);

export const use = (ctx: aether.Context): Synnax | null =>
  ctx.get<ContextValue>(CONTEXT_KEY).client;
