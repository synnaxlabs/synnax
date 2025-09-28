// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as SynnaxClient } from "@synnaxlabs/client";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useMemo,
  useRef,
} from "react";

import { Aether } from "@/aether";
import { flux } from "@/flux/aether";
import { core } from "@/flux/core";
import { useInitializerRef, useRequiredContext } from "@/hooks";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { Status } from "@/status/core";
import { Synnax } from "@/synnax";

type ContextValue = core.Client;

const Context = createContext<ContextValue | null>(null);

export const useStore = <ScopedStore extends flux.Store>(
  scope?: string,
): ScopedStore => {
  const client = useRequiredContext(Context);
  const uniqueKey = useUniqueKey(scope);
  return useMemo(() => client.scopedStore<ScopedStore>(uniqueKey), [client, uniqueKey]);
};

export type ProviderProps<ScopedStore extends flux.Store> = (
  | { client: core.Client<ScopedStore> }
  | { storeConfig: flux.StoreConfig<ScopedStore> }
) &
  PropsWithChildren;

export const Provider = <ScopedStore extends flux.Store>({
  children,
  ...cfg
}: ProviderProps<ScopedStore>): ReactElement | null => {
  const synnaxClient = Synnax.use();
  const handleError = Status.useErrorHandler();
  const handleAsyncError = Status.useAsyncErrorHandler();
  const { path } = Aether.useLifecycle({
    type: flux.PROVIDER_TYPE,
    schema: flux.providerStateZ,
    initialState: {},
  });
  const initializeClient = () => {
    if ("client" in cfg) return cfg.client;
    return new core.Client<ScopedStore>({
      client: synnaxClient,
      storeConfig: cfg.storeConfig,
      handleError,
      handleAsyncError,
    });
  };
  const clientRef = useInitializerRef(initializeClient);
  const prevSynnaxClient = useRef<SynnaxClient | null>(null);
  if (synnaxClient?.key !== prevSynnaxClient.current?.key) {
    prevSynnaxClient.current = synnaxClient;
    clientRef.current = initializeClient();
  }
  return (
    <Aether.Composite path={path}>
      <Context value={clientRef.current}>{children}</Context>
    </Aether.Composite>
  );
};
