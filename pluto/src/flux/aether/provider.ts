// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type AsyncDestructor } from "@synnaxlabs/x";
import z from "zod";

import { aether, synnax } from "@/ether";
import { createStore, type Store, type StoreConfig } from "@/flux/aether/store";
import { openStreamer } from "@/flux/aether/streamer";
import { status } from "@/status/aether";

export type ProviderState = z.input<typeof providerStateZ>;
export const providerStateZ = z.object({});

interface InternalState {
  closeStreamer: AsyncDestructor;
  store: Store;
}

export interface ContextValue {
  store: Store;
}

export const ZERO_CONTEXT_VALUE: ContextValue = {
  store: createStore({}),
};

const CONTEXT_KEY = "flux-context";

const set = (ctx: aether.Context, value: ContextValue): void =>
  ctx.set(CONTEXT_KEY, value);

export const PROVIDER_TYPE = "flux.Provider";

export const useStore = <ScopedStore extends Store>(ctx: aether.Context): ScopedStore =>
  ctx.get<ContextValue>(CONTEXT_KEY).store as ScopedStore;

const createProvider = <ScopedStore extends Store>(
  storeConfig: StoreConfig<ScopedStore>,
) => {
  const store = createStore<ScopedStore>(storeConfig);
  return class Provider extends aether.Composite<typeof providerStateZ, InternalState> {
    static readonly TYPE = PROVIDER_TYPE;
    static readonly stateZ = providerStateZ;
    schema = Provider.stateZ;

    afterUpdate(ctx: aether.Context): void {
      const { internal: i } = this;
      if (!ctx.wasSetPreviously(CONTEXT_KEY)) set(ctx, { store });
      const client = synnax.use(ctx);
      const handleError = status.useErrorHandler(ctx);
      if (client == null) return;
      handleError(async () => {
        i.closeStreamer ??= await openStreamer({
          handleError,
          storeConfig,
          client,
          store,
          openStreamer: client.openStreamer.bind(client),
        });
      });
    }
  };
};

export const createRegistry = <ScopedStore extends Store>(
  storeConfig: StoreConfig<ScopedStore>,
): aether.ComponentRegistry => ({
  [PROVIDER_TYPE]: createProvider(storeConfig),
});
