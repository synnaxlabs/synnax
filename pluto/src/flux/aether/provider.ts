// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import z from "zod";

import { aether, synnax } from "@/ether";
import { core } from "@/flux/core";
import { useAsyncErrorHandler, useErrorHandler } from "@/status/aether/aggregator";

/** State schema for the flux provider (currently empty) */
export const providerStateZ = z.object({});

/** Type representing the provider's state */
export type ProviderState = z.input<typeof providerStateZ>;

/**
 * Internal state managed by the provider.
 */
interface InternalState {
  /** The store instance */
  store: core.Client<core.Store>;
}

/**
 * Value stored in the Aether context for flux components.
 */
export type ContextValue = core.Client<core.Store>;

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
export const useStore = <ScopedStore extends core.Store>(
  ctx: aether.Context,
  scope: string,
): ScopedStore => ctx.get<ContextValue>(CONTEXT_KEY).scopedStore<ScopedStore>(scope);

export type ProviderConfig<ScopedStore extends core.Store = core.Store> =
  | {
      client: core.Client;
    }
  | {
      storeConfig: core.StoreConfig<ScopedStore>;
    };

/**
 * Creates a flux provider component class for the given store configuration.
 * The provider manages the store lifecycle and handles streamer connections.
 *
 * @template ScopedStore - The type of the store
 * @param storeConfig - Configuration for the store and its listeners
 * @returns A provider component class
 */
const createProvider = <ScopedStore extends core.Store>(
  cfg: ProviderConfig<ScopedStore>,
) => {
  const buildClient = (
    ctx: aether.Context,
    prevClient: core.Client<ScopedStore>,
  ): core.Client<ScopedStore> => {
    if ("client" in cfg) return cfg.client;
    const nextClient = synnax.use(ctx);
    if (prevClient != null && prevClient?.client?.key === nextClient?.key)
      return prevClient;
    return new core.Client<ScopedStore>({
      client: nextClient,
      storeConfig: cfg.storeConfig,
      handleError: useErrorHandler(ctx),
      handleAsyncError: useAsyncErrorHandler(ctx),
    });
  };
  return class Provider extends aether.Composite<typeof providerStateZ, InternalState> {
    static readonly TYPE = PROVIDER_TYPE;
    static readonly stateZ = providerStateZ;
    schema = Provider.stateZ;
    afterUpdate(ctx: aether.Context): void {
      const { internal: i } = this;
      i.store = buildClient(ctx, i.store);
      if (!ctx.wasSetPreviously(CONTEXT_KEY)) ctx.set(CONTEXT_KEY, i.store);
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
export const createRegistry = <ScopedStore extends core.Store>(
  cfg: ProviderConfig<ScopedStore>,
): aether.ComponentRegistry => ({ [PROVIDER_TYPE]: createProvider(cfg) });
