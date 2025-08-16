// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import {
  createContext,
  type FC,
  type PropsWithChildren,
  type ReactElement,
  use,
} from "react";

import { Flux } from "@/flux";
import { flux } from "@/flux/aether";
import { Pluto } from "@/pluto";
import { Status } from "@/status";
import { status } from "@/status/aether";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";
import { createAetherProvider } from "@/testutil/Aether";

const AetherProvider = createAetherProvider({
  ...synnax.REGISTRY,
  ...status.REGISTRY,
  ...flux.createRegistry({}),
});

interface ClientConnector {
  (connected: boolean): void;
}

const Context = createContext<ClientConnector>(() => () => {});

export const useConnectToClient = () => use(Context);

export const newSynnaxWrapper = <ScopedStore extends flux.Store>(
  client: Client | null = null,
  storeConfig?: Flux.StoreConfig<ScopedStore>,
  requireStreamerMounted = true,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <AetherProvider>
      <Status.Aggregator>
        <Synnax.TestProvider client={client}>
          <Flux.Provider
            storeConfig={
              storeConfig ??
              (Pluto.FLUX_STORE_CONFIG as unknown as Flux.StoreConfig<ScopedStore>)
            }
            requireStreamerMounted={requireStreamerMounted}
          >
            {children}
          </Flux.Provider>
        </Synnax.TestProvider>
      </Status.Aggregator>
    </AetherProvider>
  );
  return Wrapper;
};
