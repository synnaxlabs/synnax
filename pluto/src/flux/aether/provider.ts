// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cache, type dispatch } from "@synnaxlabs/client";
import z from "zod";

import { aether } from "@/aether/aether";
import { base } from "@/flux/base";
import { useAsyncErrorHandler, useErrorHandler } from "@/status/aether/aggregator";
import { synnax } from "@/synnax/aether";

/** State schema for the flux provider (currently empty) */
export const providerStateZ = z.object({});

/** Type representing the provider's state */
export type ProviderState = z.input<typeof providerStateZ>;

/**
 * Value stored in the Aether context for flux components.
 */
export interface ContextValue {
  engine: cache.Engine;
  controllers: Record<string, dispatch.Controller<any, any, any>>;
}

interface InternalState {
  value: ContextValue;
  clientKey: string | undefined;
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
export const useStore = <ScopedStore extends base.Store>(
  ctx: aether.Context,
  scope: string,
): ScopedStore => {
  const { engine, controllers } = ctx.get<ContextValue>(CONTEXT_KEY);
  return base.createScopedStore<ScopedStore>(engine, controllers, scope);
};

export interface ProviderConfig {
  /** Builds the dispatch controller overlaying each undoable store key. */
  composers?: base.StoreComposers;
}

/**
 * Creates a flux provider component class. The provider binds the connected
 * client's cache engine (or a detached engine when disconnected) into the
 * Aether context and starts change streaming.
 */
const createProvider = (cfg: ProviderConfig = {}) =>
  class Provider extends aether.Composite<typeof providerStateZ, InternalState> {
    static readonly TYPE = PROVIDER_TYPE;
    static readonly stateZ = providerStateZ;
    schema = Provider.stateZ;

    afterUpdate(ctx: aether.Context): void {
      const { internal: i } = this;
      const client = synnax.use(ctx);
      const stale = i.value == null || client?.key !== i.clientKey;
      if (stale) {
        i.clientKey = client?.key;
        const attachedClient = client?.cache.enabled === true ? client : null;
        const engine =
          attachedClient?.cache.engine ?? new cache.Engine({ openStreamer: null });
        const handleError = useErrorHandler(ctx);
        engine.setErrorHandlers(handleError, useAsyncErrorHandler(ctx));
        const controllers = Object.fromEntries(
          Object.entries(cfg.composers ?? {}).map(([key, compose]) => [
            key,
            compose({ client: attachedClient, engine }),
          ]),
        );
        handleError(
          async () => await engine.ensureStreaming(),
          "failed to start flux change streaming",
        );
        i.value = { engine, controllers };
      }
      if (stale || !ctx.wasSetPreviously(CONTEXT_KEY)) ctx.set(CONTEXT_KEY, i.value);
    }
  };

/**
 * Creates an Aether component registry with the flux provider.
 *
 * @param cfg - Optional provider configuration
 * @returns An Aether component registry containing the provider
 */
export const createRegistry = (cfg?: ProviderConfig): aether.ComponentRegistry => ({
  [PROVIDER_TYPE]: createProvider(cfg),
});
