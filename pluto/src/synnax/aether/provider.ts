// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax, synnaxParamsZ } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { useErrorHandler } from "@/status/aether/aggregator";

const stateZ = z.object({
  props: synnaxParamsZ.nullable(),
  state: Synnax.connectivity.connectionStateZ.nullable(),
});

export interface ContextValue {
  client: Synnax | null;
}

export const ZERO_CONTEXT_VALUE: ContextValue = {
  client: null,
};

export class Provider extends aether.Composite<typeof stateZ, ContextValue> {
  static readonly TYPE = "synnax.Provider";
  static readonly stateZ = stateZ;
  schema = Provider.stateZ;

  afterUpdate(ctx: aether.Context): void {
    if (!ctx.wasSetPreviously(CONTEXT_KEY)) set(ctx, ZERO_CONTEXT_VALUE);
    if (this.state.props == null) {
      this.closeClient();
      set(ctx, this.internal);
      return;
    }

    if (
      this.prevState.props != null &&
      deep.equal(this.state.props, this.prevState.props) &&
      this.internal.client != null
    )
      return;

    this.closeClient();
    this.internal.client = new Synnax({
      ...this.state.props,
      onInternalError: useErrorHandler(ctx),
    });
    set(ctx, this.internal);
  }

  afterDelete(): void {
    this.closeClient();
  }

  private closeClient(): void {
    if (this.internal.client == null) return;
    this.internal.client.close();
    this.internal.client = null;
  }
}

const CONTEXT_KEY = "pluto-client-context";

const set = (ctx: aether.Context, value: ContextValue): void =>
  ctx.set(CONTEXT_KEY, value);

export const use = (ctx: aether.Context): Synnax | null =>
  ctx.get<ContextValue>(CONTEXT_KEY)?.client ?? null;

export const REGISTRY: aether.ComponentRegistry = {
  [Provider.TYPE]: Provider,
};
