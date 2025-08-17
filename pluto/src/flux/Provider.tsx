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
  useState,
} from "react";

import { Aether } from "@/aether";
import { flux } from "@/flux/aether";
import { scopeStore } from "@/flux/aether/store";
import { useAsyncEffect, useInitializerRef, useRequiredContext } from "@/hooks";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

interface ContextValue {
  listenersMounted: boolean;
  store: flux.InternalStore;
}

const Context = createContext<ContextValue | null>(null);

export const useStore = <ScopedStore extends flux.Store>(scope?: string): ScopedStore =>
  scopeStore<ScopedStore>(useRequiredContext(Context)?.store, useUniqueKey(scope));

export interface ProviderProps<ScopedStore extends flux.Store>
  extends PropsWithChildren {
  openStreamer?: framer.StreamOpener;
  storeConfig?: flux.StoreConfig<ScopedStore>;
  requireStreamerMounted?: boolean;
}

export const Provider = <ScopedStore extends flux.Store>({
  children,
  openStreamer: propsOpenStreamer,
  storeConfig = {} as flux.StoreConfig<ScopedStore>,
  requireStreamerMounted = false,
}: ProviderProps<ScopedStore>): ReactElement | null => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const storeRef = useInitializerRef<flux.InternalStore>(() =>
    flux.createStore<ScopedStore>(storeConfig, handleError),
  );
  const [streamerMounted, setStreamerMounted] = useState(!requireStreamerMounted);
  const { path } = Aether.useLifecycle({
    type: flux.PROVIDER_TYPE,
    schema: flux.providerStateZ,
    initialState: {},
  });

  const openStreamer = useMemo(
    () => propsOpenStreamer ?? client?.openStreamer.bind(client),
    [client, propsOpenStreamer],
  );

  useAsyncEffect(async () => {
    if (openStreamer == null || client == null) return;
    const destructor = await flux.openStreamer({
      handleError,
      storeConfig,
      client,
      store: scopeStore<ScopedStore>(storeRef.current, ""),
      openStreamer,
    });
    setStreamerMounted(true);
    return () => destructor();
  }, [openStreamer]);
  const value = useMemo(
    () => ({
      listenersMounted: streamerMounted,
      store: storeRef.current,
    }),
    [streamerMounted, storeRef.current],
  );
  return (
    <Aether.Composite path={path}>
      <Context value={value}>{children}</Context>
    </Aether.Composite>
  );
};
