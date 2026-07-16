// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cache, type dispatch } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement, useMemo, useRef } from "react";

import { Aether } from "@/aether";
import { context } from "@/context";
import { flux } from "@/flux/aether";
import { base } from "@/flux/base";
import { useInitializerRef } from "@/hooks";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { type state } from "@/state";
import { type status } from "@/status/aether";
import { Status } from "@/status/base";
import { Synnax } from "@/synnax";

interface ContextValue {
  engine: cache.Engine;
  controllers: Record<string, dispatch.Controller<any, any, any>>;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "Flux.Context",
  providerName: "Flux.Provider",
});

export const useStore = <ScopedStore extends flux.Store>(
  scope?: string,
): ScopedStore => {
  const { engine, controllers } = useContext("Flux.useStore");
  const uniqueKey = useUniqueKey(scope);
  return useMemo(
    () => base.createScopedStore<ScopedStore>(engine, controllers, uniqueKey),
    [engine, controllers, uniqueKey],
  );
};

/// Returns the typed query cache for the given key, creating it on first use.
/// Each `createRetrieve` operation owns a unique key; the returned cache has
/// concrete `Q` and `D` types and requires no per-call generics or casts.
export const useQueryCache = <Q extends base.Query, D extends state.State>(
  key: string,
): cache.QueryCache<Q, D> =>
  useContext("Flux.useQueryCache").engine.getCache<Q, D>(key);

export interface ProviderProps extends PropsWithChildren {
  /**
   * Builds the dispatch controller overlaying each undoable store key.
   * Assembled at the Pluto wiring site; must be referentially stable.
   */
  composers?: base.StoreComposers;
  /**
   * Overrides the engine backing the flux stores. Defaults to the connected
   * client's cache engine, or a detached engine when disconnected.
   */
  engine?: cache.Engine;
  /**
   * Overrides error reporting for store listeners and change streaming.
   * Defaults to the status aggregator.
   */
  handleError?: status.ErrorHandler;
  /** Async counterpart of handleError. */
  handleAsyncError?: status.AsyncErrorHandler;
}

const NO_COMPOSERS: base.StoreComposers = {};

export const Provider = ({
  children,
  composers = NO_COMPOSERS,
  engine: engineOverride,
  handleError: handleErrorOverride,
  handleAsyncError: handleAsyncErrorOverride,
}: ProviderProps): ReactElement | null => {
  const synnaxClient = Synnax.use();
  const aggregatorHandleError = Status.useErrorHandler();
  const aggregatorHandleAsyncError = Status.useAsyncErrorHandler();
  const handleError = handleErrorOverride ?? aggregatorHandleError;
  const handleAsyncError = handleAsyncErrorOverride ?? aggregatorHandleAsyncError;
  const { path } = Aether.useLifecycle({
    type: flux.PROVIDER_TYPE,
    schema: flux.providerStateZ,
    initialState: {},
  });
  const initialize = (): ContextValue => {
    const attachedClient =
      engineOverride == null && synnaxClient?.cache.enabled === true
        ? synnaxClient
        : null;
    const engine =
      engineOverride ??
      attachedClient?.cache.engine ??
      new cache.Engine({ openStreamer: null });
    engine.setErrorHandlers(handleError, handleAsyncError);
    const controllers = Object.fromEntries(
      Object.entries(composers).map(([key, compose]) => [
        key,
        compose({ client: attachedClient, engine }),
      ]),
    );
    handleError(
      async () => await engine.ensureStreaming(),
      "failed to start flux change streaming",
    );
    return { engine, controllers };
  };
  const valueRef = useInitializerRef(initialize);
  const prevClientKey = useRef(synnaxClient?.key);
  if (synnaxClient?.key !== prevClientKey.current) {
    prevClientKey.current = synnaxClient?.key;
    valueRef.current = initialize();
  }
  return (
    <Aether.Composite path={path}>
      <Context value={valueRef.current}>{children}</Context>
    </Aether.Composite>
  );
};
