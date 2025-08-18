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
  ...flux.createRegistry({ storeConfig: {} }),
});

interface ClientConnector {
  (connected: boolean): void;
}

const Context = createContext<ClientConnector>(() => () => {});

export const useConnectToClient = () => use(Context);

export const newSynnaxWrapper = (
  client: Client | null = null,
): FC<PropsWithChildren> => {
  const fluxClient = new Flux.Client({
    client,
    storeConfig: Pluto.FLUX_STORE_CONFIG,
    handleError: status.createErrorHandler(console.error),
    handleAsyncError: status.createAsyncErrorHandler(console.error),
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <AetherProvider>
      <Status.Aggregator>
        <Synnax.TestProvider client={client}>
          <Flux.Provider client={fluxClient}>{children}</Flux.Provider>
        </Synnax.TestProvider>
      </Status.Aggregator>
    </AetherProvider>
  );
  return Wrapper;
};

export const newSynnaxWrapperWithAwait = async (
  client: Client | null = null,
): Promise<FC<PropsWithChildren>> => {
  const fluxClient = new Flux.Client({
    client,
    storeConfig: Pluto.FLUX_STORE_CONFIG,
    handleError: status.createErrorHandler(console.error),
    handleAsyncError: status.createAsyncErrorHandler(console.error),
  });
  await fluxClient.awaitInitialized();
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <AetherProvider>
      <Status.Aggregator>
        <Synnax.TestProvider client={client}>
          <Flux.Provider client={fluxClient}>{children}</Flux.Provider>
        </Synnax.TestProvider>
      </Status.Aggregator>
    </AetherProvider>
  );
  return Wrapper;
};
