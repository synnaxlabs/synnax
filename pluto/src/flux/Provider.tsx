// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type framer } from "@synnaxlabs/client";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useMemo,
} from "react";

import { Aether } from "@/aether";
import { flux } from "@/flux/aether";
import { scopeStore } from "@/flux/aether/store";
import { useAsyncEffect, useInitializerRef, useRequiredContext } from "@/hooks";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

interface ContextValue {
  mounted: Promise<void>;
  store: flux.InternalStore;
}

const Context = createContext<ContextValue | null>(null);

export interface UseStoreReturn<ScopedStore extends flux.Store> {
  store: ScopedStore;
  mounted: Promise<void>;
}

export const useStore = <ScopedStore extends flux.Store>(
  scope?: string,
): UseStoreReturn<ScopedStore> => {
  const { store, mounted } = useRequiredContext(Context);
  return { store: scopeStore<ScopedStore>(store, useUniqueKey(scope)), mounted };
};

export interface ProviderProps<ScopedStore extends flux.Store>
  extends PropsWithChildren {
  openStreamer?: framer.StreamOpener;
  storeConfig?: flux.StoreConfig<ScopedStore>;
}

export const Provider = <ScopedStore extends flux.Store>({
  children,
  openStreamer: propsOpenStreamer,
  storeConfig = {} as flux.StoreConfig<ScopedStore>,
}: ProviderProps<ScopedStore>): ReactElement | null => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const storeRef = useInitializerRef<flux.InternalStore>(() =>
    flux.createStore<ScopedStore>(storeConfig, handleError),
  );
  const { path } = Aether.useLifecycle({
    type: flux.PROVIDER_TYPE,
    schema: flux.providerStateZ,
    initialState: {},
  });

  const destructor = useMemo(() => {
    if (client == null) return Promise.resolve(null);
    return flux.openStreamer({
      handleError,
      storeConfig,
      client,
      store: scopeStore<ScopedStore>(storeRef.current, ""),
      openStreamer: propsOpenStreamer ?? client?.openStreamer.bind(client),
    });
  }, [client, propsOpenStreamer]);
  useAsyncEffect(
    async () => async () => {
      const d = await destructor;
      if (d == null) return;
      await d();
    },
    [destructor],
  );

  const value = useMemo(
    (): ContextValue => ({
      store: storeRef.current,
      mounted: (async () => {
        await destructor;
      })(),
    }),
    [storeRef.current, destructor],
  );
  return (
    <Aether.Composite path={path}>
      <Context value={value}>{children}</Context>
    </Aether.Composite>
  );
};
