// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { synnax } from "@/synnax/aether";
import { Context, CONTEXT_KEY, setContext } from "@/telem/aether/context";
import { CompoundFactory, createFactory, type Factory } from "@/telem/aether/factory";
import { NoopFactory } from "@/telem/aether/noop";
import { PipelineFactory } from "@/telem/aether/pipeline";
import { type Client, RemoteFactory } from "@/telem/aether/remote";
import { StaticFactory } from "@/telem/aether/static";
import { TransformerFactory } from "@/telem/aether/transformers";

export type ProviderState = z.input<typeof providerStateZ>;
export const providerStateZ = z.object({});

export const PROVIDER_TYPE = "telem.Provider";

export const createProvider = (
  createFactory: (client: Client | null) => CompoundFactory,
): aether.ComponentConstructor => {
  class BaseProvider extends aether.Composite<typeof providerStateZ> {
    static readonly TYPE = PROVIDER_TYPE;
    static readonly stateZ = providerStateZ;
    schema = BaseProvider.stateZ;
    prevCore: Synnax | null = null;

    afterUpdate(ctx: aether.Context): void {
      const core = synnax.use(ctx);
      const shouldSwap = core !== this.prevCore || !ctx.wasSetPreviously(CONTEXT_KEY);
      if (!shouldSwap) return;
      this.prevCore = core;
      setContext(ctx, new Context(createFactory(core)));
    }
  }
  return BaseProvider;
};

export const Provider = createProvider(createFactory);

export const REGISTRY: aether.ComponentRegistry = {
  [PROVIDER_TYPE]: Provider,
};

export type FactoryConstructor = (client: Client | null) => Factory;

export const createRegistry = (
  ...factoryConstructors: FactoryConstructor[]
): aether.ComponentRegistry => {
  const create = (client: Client | null): CompoundFactory => {
    const base = [new TransformerFactory(), new StaticFactory(), new NoopFactory()];
    const f = new CompoundFactory(base);
    f.add(new RemoteFactory(client));
    for (const constructor of factoryConstructors) f.add(constructor(client));
    f.add(new PipelineFactory(f));
    return f;
  };
  return { [PROVIDER_TYPE]: createProvider(create) };
};
