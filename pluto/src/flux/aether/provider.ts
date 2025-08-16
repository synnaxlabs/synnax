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

/** State schema for the flux provider (currently empty) */
export const providerStateZ = z.object({});

/** Type representing the provider's state */
export type ProviderState = z.input<typeof providerStateZ>;

/**
 * Internal state managed by the provider.
 */
interface InternalState {
  /** Function to close the active streamer connection */
  closeStreamer: AsyncDestructor;
  /** The store instance */
  store: Store;
}

/**
 * Value stored in the Aether context for flux components.
 */
export interface ContextValue {
  /** The store instance available to child components */
  store: Store;
}

/**
 * Default context value with an empty store.
 * Used as a fallback when no provider is present.
 */
export const ZERO_CONTEXT_VALUE: ContextValue = {
  store: createStore({}),
};

/** Key used to store flux context in the Aether context */
const CONTEXT_KEY = "flux-context";

/**
 * Sets the flux context value in the Aether context.
 * 
 * @param ctx - The Aether context
 * @param value - The context value to set
 */
const set = (ctx: aether.Context, value: ContextValue): void =>
  ctx.set(CONTEXT_KEY, value);

/** Type identifier for the flux provider component */
export const PROVIDER_TYPE = "flux.Provider";

/**
 * Hook to access the flux store from the Aether context.
 * 
 * @template ScopedStore - The type of the store
 * @param ctx - The Aether context
 * @returns The store instance from the context
 */
export const useStore = <ScopedStore extends Store>(ctx: aether.Context): ScopedStore =>
  ctx.get<ContextValue>(CONTEXT_KEY).store as ScopedStore;

/**
 * Creates a flux provider component class for the given store configuration.
 * The provider manages the store lifecycle and handles streamer connections.
 * 
 * @template ScopedStore - The type of the store
 * @param storeConfig - Configuration for the store and its listeners
 * @returns A provider component class
 */
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

/**
 * Creates an Aether component registry with the flux provider.
 * 
 * @template ScopedStore - The type of the store
 * @param storeConfig - Configuration for the store and its listeners
 * @returns An Aether component registry containing the provider
 */
export const createRegistry = <ScopedStore extends Store>(
  storeConfig: StoreConfig<ScopedStore>,
): aether.ComponentRegistry => ({
  [PROVIDER_TYPE]: createProvider(storeConfig),
});
