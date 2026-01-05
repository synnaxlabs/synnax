// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type FC, type PropsWithChildren, type ReactElement } from "react";

import { aetherTest } from "@/aether/test";
import { Flux } from "@/flux";
import { flux } from "@/flux/aether";
import { Pluto } from "@/pluto";
import { status } from "@/status/aether";
import { Status } from "@/status/core";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";

const AetherProvider = aetherTest.createProvider({
  ...synnax.REGISTRY,
  ...status.REGISTRY,
  ...flux.createRegistry({ storeConfig: {} }),
});

const newWrapper = (client: Client | null, fluxClient: Flux.Client) => {
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

export interface CreateSynnaxWrapperArgs {
  client: Client | null;
  excludeFluxStores?: string[];
}

const createFluxClient = (args: CreateSynnaxWrapperArgs): Flux.Client => {
  const { client, excludeFluxStores } = args;
  const storeConfig = { ...Pluto.FLUX_STORE_CONFIG };
  if (excludeFluxStores)
    excludeFluxStores.forEach((store) => delete storeConfig[store]);
  return new Flux.Client({
    client,
    storeConfig,
    handleError: status.createErrorHandler(console.error),
    handleAsyncError: status.createAsyncErrorHandler(console.error),
  });
};

export const createSynnaxWrapper = ({
  client,
  excludeFluxStores,
}: CreateSynnaxWrapperArgs): FC<PropsWithChildren> =>
  newWrapper(client, createFluxClient({ client, excludeFluxStores }));

export const createAsyncSynnaxWrapper = async (
  args: CreateSynnaxWrapperArgs,
): Promise<FC<PropsWithChildren>> => {
  const { client } = args;
  const fluxClient = createFluxClient(args);
  await fluxClient.awaitInitialized();
  return newWrapper(client, fluxClient);
};
