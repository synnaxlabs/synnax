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

import { AetherComposite, AetherContext } from "@/core/aether/worker";

export namespace AetherClient {
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

  export class Provider extends AetherComposite<typeof stateZ, ContextValue> {
    static readonly TYPE = "ClientProvider";
    static readonly stateZ = stateZ;
    schema = Provider.stateZ;

    afterUpdate(): void {
      console.log(this.state.props);
      if (this.ctx.getOptional(Context.CONTEXT_KEY) == null) {
        Context.set(this.ctx, ZERO_CONTEXT_VALUE);
      }
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
        console.log("RETURN", this.prevState.props);
        return;
      }

      this.internal.client = new Synnax(this.state.props);
      this.internal.client.connectivity.onChange((state) =>
        this.setState((p) => ({ ...p, state }))
      );
      console.log("SET");
      Context.set(this.ctx, this.internal);
    }
  }

  class Context {
    static readonly CONTEXT_KEY = "pluto-client-context";

    value: ContextValue = ZERO_CONTEXT_VALUE;

    private constructor(value: ContextValue) {
      this.value = value;
    }

    static set(ctx: AetherContext, value: ContextValue): void {
      const telem = new Context(value);
      ctx.set(Context.CONTEXT_KEY, telem);
    }

    static use(ctx: AetherContext): Synnax | null {
      return ctx.get<Context>(Context.CONTEXT_KEY).value.client;
    }
  }

  export const use = Context.use;
}
