// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import z from "zod";

import { aether } from "@/aether/aether";
import { base } from "@/flux/base";
import { useAsyncErrorHandler, useErrorHandler } from "@/status/aether/aggregator";
import { type ErrorHandler } from "@/status/aether/errorHandler";
import { synnax } from "@/synnax/aether";

/** State schema for the flux provider (currently empty) */
export const providerStateZ = z.object({});

/** Type representing the provider's state */
export type ProviderState = z.input<typeof providerStateZ>;

/**
 * Internal state managed by the provider.
 */
interface InternalState {
  /** The store instance */
  store: base.Client<base.Store>;
  /** Error handler bound from the current afterUpdate's context, reused by afterDelete
   * (which has no update-triggering context of its own to derive one from). */
  runAsync: ErrorHandler;
}

/**
 * Value stored in the Aether context for flux components.
 */
export type ContextValue = base.Client<base.Store>;

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
export const useStore = <ScopedStore extends base.Store>(
  ctx: aether.Context,
  scope: string,
): ScopedStore => ctx.get<ContextValue>(CONTEXT_KEY).scopedStore<ScopedStore>(scope);

export type ProviderConfig<ScopedStore extends base.Store = base.Store> =
  | {
      client: base.Client;
    }
  | {
      storeConfig: base.StoreConfig<ScopedStore>;
    };

/**
 * Creates a flux provider component class for the given store configuration.
 * The provider manages the store lifecycle and handles streamer connections.
 *
 * @template ScopedStore - The type of the store
 * @param storeConfig - Configuration for the store and its listeners
 * @returns A provider component class
 */
const createProvider = <ScopedStore extends base.Store>(
  cfg: ProviderConfig<ScopedStore>,
) => {
  // A caller-supplied client (the `client` variant of ProviderConfig) is owned by the
  // caller, which is responsible for closing it. Only a client this provider builds
  // itself (the `storeConfig` variant) is closed here.
  const owns = !("client" in cfg);

  return class Provider extends aether.Composite<typeof providerStateZ, InternalState> {
    static readonly TYPE = PROVIDER_TYPE;
    static readonly stateZ = providerStateZ;
    schema = Provider.stateZ;

    afterUpdate(ctx: aether.Context): void {
      const { internal: i } = this;
      if (owns) i.runAsync = useErrorHandler(ctx);
      i.store = this.buildClient(ctx, i.store);
      if (!ctx.wasSetPreviously(CONTEXT_KEY)) ctx.set(CONTEXT_KEY, i.store);
    }

    afterDelete(): void {
      this.closeClient(this.internal.store);
    }

    private buildClient(
      ctx: aether.Context,
      prevClient: base.Client<ScopedStore>,
    ): base.Client<ScopedStore> {
      if ("client" in cfg) return cfg.client;
      const nextClient = synnax.use(ctx);
      if (prevClient != null && prevClient?.client?.key === nextClient?.key)
        return prevClient;
      this.closeClient(prevClient);
      return new base.Client<ScopedStore>({
        client: nextClient,
        storeConfig: cfg.storeConfig,
        handleError: this.internal.runAsync,
        handleAsyncError: useAsyncErrorHandler(ctx),
      });
    }

    private closeClient(prevClient: base.Client<ScopedStore> | undefined): void {
      if (!owns || prevClient == null) return;
      this.internal.runAsync(async () => {
        await prevClient.close();
      }, "failed to close flux client");
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
export const createRegistry = <ScopedStore extends base.Store>(
  cfg: ProviderConfig<ScopedStore>,
): aether.ComponentRegistry => ({ [PROVIDER_TYPE]: createProvider(cfg) });
