// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Instrumentation } from "@synnaxlabs/alamos";
import { type Synnax } from "@synnaxlabs/client";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { Context, CONTEXT_KEY, setContext } from "@/telem/aether/context";
import { type CompoundFactory, createFactory } from "@/telem/aether/factory";
import { client } from "@/telem/client";

export type ProviderState = z.input<typeof providerStateZ>;
export const providerStateZ = z.object({});

interface InternalState {
  instrumentation: Instrumentation;
}

export const PROVIDER_TYPE = "telem.Provider";

export const createProvider = (
  createFactory: (client?: client.Client) => CompoundFactory,
): aether.ComponentConstructor => {
  class BaseProvider extends aether.Composite<typeof providerStateZ, InternalState> {
    static readonly TYPE = PROVIDER_TYPE;
    static readonly stateZ = providerStateZ;
    schema = BaseProvider.stateZ;
    prevCore: Synnax | null = null;
    client: client.Client | null = null;

    afterUpdate(ctx: aether.Context): void {
      const { internal: i } = this;
      const core = synnax.use(ctx);
      const runAsync = status.useErrorHandler(ctx);
      i.instrumentation = alamos.useInstrumentation(ctx, "telem").child("provider");
      const shouldSwap = core !== this.prevCore || !ctx.wasSetPreviously(CONTEXT_KEY);
      if (!shouldSwap) return;
      this.prevCore = core;
      if (this.client != null)
        runAsync(async () => {
          if (this.client == null) throw new Error("no client to close");
          await this.client.close();
        }, "failed to close client");

      this.client =
        core == null
          ? new client.NoopClient()
          : new client.Core({ core, instrumentation: i.instrumentation });
      const f = createFactory(this.client);
      const value = new Context(f);
      setContext(ctx, value);
    }
  }
  return BaseProvider;
};

export const Provider = createProvider(createFactory);

export const REGISTRY: aether.ComponentRegistry = {
  [PROVIDER_TYPE]: Provider,
};
