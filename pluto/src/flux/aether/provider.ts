// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { type AsyncDestructor } from "@synnaxlabs/x";
import z from "zod";

import { aether, synnax } from "@/ether";
import {
  createStore,
  type InternalStore,
  scopeStore,
  type Store,
  type StoreConfig,
} from "@/flux/aether/store";
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
  store: InternalStore;
  /** The Synnax client instance */
  client: Synnax | null;
}

/**
 * Value stored in the Aether context for flux components.
 */
export interface ContextValue {
  /** The store instance available to child components */
  store: InternalStore;
}

/** Key used to store flux context in the Aether context */
const CONTEXT_KEY = "flux-context";

/** Type identifier for the flux provider component */
export const PROVIDER_TYPE = "flux.Provider";

/**
 * Hook to access the flux store from the Aether context.
 *
 * @template ScopedStore - The type of the store
 * @param ctx - The Aether context
 * @returns The store instance from the context
 */
export const useStore = <ScopedStore extends Store>(
  ctx: aether.Context,
  scope: string,
): ScopedStore =>
  scopeStore<ScopedStore>(ctx.get<ContextValue>(CONTEXT_KEY).store, scope);

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
) =>
  class Provider extends aether.Composite<typeof providerStateZ, InternalState> {
    static readonly TYPE = PROVIDER_TYPE;
    static readonly stateZ = providerStateZ;
    schema = Provider.stateZ;

    afterUpdate(ctx: aether.Context): void {
      const { internal: i } = this;
      const handleError = status.useErrorHandler(ctx);
      if (!ctx.wasSetPreviously(CONTEXT_KEY)) {
        i.store = createStore<ScopedStore>(storeConfig, handleError);
        ctx.set(CONTEXT_KEY, { store: i.store });
      }
      const nextClient = synnax.use(ctx);
      if (i.client?.key === nextClient?.key) return;
      // This means we've either switched connections or disconnected. In either case,
      // we need to clear the store, stop the streamer, and start a new one (if connected).
      i.client = nextClient;

      handleError(async () => {
        await i.closeStreamer?.();
        if (i.client == null) return;
        Object.values(i.store).forEach((store) => store.clear());
        i.closeStreamer = await openStreamer({
          handleError,
          storeConfig,
          client: i.client,
          store: scopeStore<ScopedStore>(i.store, ""),
          openStreamer: i.client.openStreamer.bind(i.client),
        });
      });
    }
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
