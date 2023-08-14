// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax, synnaxPropsZ } from "@synnaxlabs/client";
import { z } from "zod";

import { AetherComposite, AetherContext } from "@/core/aether/worker";

export namespace AetherClient {
  const aetherClientProviderState = z.object({
    props: synnaxPropsZ.optional(),
  });

  interface InternalState {
    client?: Synnax;
  }

  export class Provider extends AetherComposite<
    typeof aetherClientProviderState,
    InternalState
  > {
    static readonly TYPE = "ClientProvider";
    static readonly stateZ = aetherClientProviderState;
    schema = Provider.stateZ;

    afterUpdate(): void {
      this.internal.client?.close();
      if (this.state.props == null) {
        this.internal.client = undefined;
      } else {
        this.internal.client = new Synnax(this.state.props);
      }
    }
  }

  export class Context {
    private static readonly CONTEXT_KEY = "pluto-client-context";

    client: Synnax;

    private constructor(client: Synnax) {
      this.client = client;
    }

    static set(ctx: AetherContext, client: Synnax): void {
      const telem = new Context(client);
      ctx.set(Context.CONTEXT_KEY, telem);
    }

    static use(ctx: AetherContext): Synnax {
      return ctx.get<Context>(Context.CONTEXT_KEY).client;
    }
  }
}
