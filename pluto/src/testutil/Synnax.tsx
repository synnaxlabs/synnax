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

import { aether } from "@/aether/aether";
import { aetherTest } from "@/aether/test";
import { Flux } from "@/flux";
import { flux } from "@/flux/aether";
import { Pluto } from "@/pluto";
import { status } from "@/status/aether";
import { Status } from "@/status/base";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";

const newWrapper = (
  client: Client | null,
  fluxClient: Flux.Client,
  additionalRegistry?: aether.ComponentRegistry,
) => {
  const AetherProvider = aetherTest.createProvider({
    ...synnax.REGISTRY,
    ...status.REGISTRY,
    ...flux.createRegistry({ storeConfig: {} }),
    ...additionalRegistry,
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

export interface CreateSynnaxWrapperArgs {
  client: Client | null;
  excludeFluxStores?: string[];
  /** Overrides the flux error handler. Defaults to logging via console.error. */
  handleError?: status.ErrorHandler;
  /** Overrides the flux async error handler. Defaults to logging via console.error. */
  handleAsyncError?: status.AsyncErrorHandler;
  /** Extra aether components merged into the test render registry. */
  additionalRegistry?: aether.ComponentRegistry;
}

const createFluxClient = (args: CreateSynnaxWrapperArgs): Flux.Client => {
  const { client, excludeFluxStores, handleError, handleAsyncError } = args;
  const storeConfig = { ...Pluto.FLUX_STORE_CONFIG };
  if (excludeFluxStores)
    excludeFluxStores.forEach((store) => delete storeConfig[store]);
  return new Flux.Client({
    client,
    storeConfig,
    handleError: handleError ?? status.createErrorHandler(console.error),
    handleAsyncError: handleAsyncError ?? status.createAsyncErrorHandler(console.error),
  });
};

export const createSynnaxWrapper = (
  args: CreateSynnaxWrapperArgs,
): FC<PropsWithChildren> =>
  newWrapper(args.client, createFluxClient(args), args.additionalRegistry);

export const createAsyncSynnaxWrapper = async (
  args: CreateSynnaxWrapperArgs,
): Promise<FC<PropsWithChildren>> => {
  const fluxClient = createFluxClient(args);
  await fluxClient.awaitInitialized();
  return newWrapper(args.client, fluxClient, args.additionalRegistry);
};
