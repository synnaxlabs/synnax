// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers, configureStore, type EnhancedStore } from "@reduxjs/toolkit";
import { type Synnax as Client } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Aether, Flux, Pluto, Status, Synnax } from "@synnaxlabs/pluto";
import { aether, flux, status, synnax } from "@synnaxlabs/pluto/ether";
import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement, useMemo } from "react";
import { Provider } from "react-redux";

import { Cluster } from "@/cluster";
import { Session } from "@/layered/session";
import { Layout } from "@/layout";
import { Project } from "@/project";

const consoleReducer = combineReducers({
  [Layout.SLICE_NAME]: Layout.reducer,
  [Drift.SLICE_NAME]: Drift.reducer,
  [Session.Log.SLICE_NAME]: Session.Log.reducer,
  [Project.SLICE_NAME]: Project.reducer,
  [Cluster.SLICE_NAME]: Cluster.reducer,
});

export type ConsolePreloadedState = {
  [Layout.SLICE_NAME]?: Layout.SliceState;
  [Session.Log.SLICE_NAME]?: Session.Log.SliceState;
  [Project.SLICE_NAME]?: Project.SliceState;
  [Cluster.SLICE_NAME]?: Cluster.SliceState;
};

export interface ConsoleTestProviderOptions {
  preloadedState?: ConsolePreloadedState;
}

export const createTestStore = (
  options: ConsoleTestProviderOptions = {},
): EnhancedStore => {
  const { preloadedState } = options;
  return configureStore({
    reducer: consoleReducer,
    preloadedState,
  });
};

const AETHER_REGISTRY: aether.ComponentRegistry = {
  ...synnax.REGISTRY,
  ...status.REGISTRY,
  ...flux.createRegistry({ storeConfig: {} }),
};

const AetherTestProvider = ({ children }: PropsWithChildren): ReactElement => {
  const worker = useMemo(() => {
    const [workerSide, mainSide] = aether.createMockPair();
    aether.render({ worker: workerSide, registry: AETHER_REGISTRY });
    return mainSide;
  }, []);
  return <Aether.Provider worker={worker}>{children}</Aether.Provider>;
};

const createFluxClient = (client: Client | null): Flux.Client =>
  new Flux.Client({
    client,
    storeConfig: { ...Pluto.FLUX_STORE_CONFIG },
    handleError: status.createErrorHandler(console.error),
    handleAsyncError: status.createAsyncErrorHandler(console.error),
  });

export const ConsoleTestProvider = ({
  store,
  client,
  fluxClient,
  children,
}: PropsWithChildren<{
  store: EnhancedStore;
  client: Client | null;
  fluxClient: Flux.Client;
}>): ReactElement => (
  <AetherTestProvider>
    <Status.Aggregator>
      <Synnax.TestProvider client={client}>
        <Flux.Provider client={fluxClient}>
          <Provider store={store}>{children}</Provider>
        </Flux.Provider>
      </Synnax.TestProvider>
    </Status.Aggregator>
  </AetherTestProvider>
);

export interface RenderWithConsoleOptions extends RenderOptions {
  preloadedState?: ConsolePreloadedState;
  store?: EnhancedStore;
}

export const renderWithConsole = (
  ui: ReactElement,
  options: RenderWithConsoleOptions = {},
): RenderResult & { store: EnhancedStore } => {
  const {
    preloadedState,
    store = createTestStore({ preloadedState }),
    ...rest
  } = options;
  const Wrapper = ({ children }: PropsWithChildren) => (
    <ConsoleTestProvider
      store={store}
      client={null}
      fluxClient={createFluxClient(null)}
    >
      {children}
    </ConsoleTestProvider>
  );
  return { ...render(ui, { wrapper: Wrapper, ...rest }), store };
};

export interface CreateConsoleWrapperArgs {
  client: Client | null;
  preloadedState?: ConsolePreloadedState;
  store?: EnhancedStore;
}

/**
 * Builds a provider wrapper backed by a real Synnax client for live-core tests. The
 * returned wrapper mounts the same provider stack as renderWithConsole (Aether, status,
 * Synnax, flux, and the Redux store) but routes flux retrieves and dispatches through
 * the given client, so components exercise the production query infrastructure against
 * a running cluster rather than reading pre-populated state. Awaits flux store
 * initialization before returning so listeners are live.
 */
export const createConsoleWrapper = async ({
  client,
  preloadedState,
  store = createTestStore({ preloadedState }),
}: CreateConsoleWrapperArgs): Promise<{
  wrapper: FC<PropsWithChildren>;
  store: EnhancedStore;
}> => {
  const fluxClient = createFluxClient(client);
  await fluxClient.awaitInitialized();
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <ConsoleTestProvider store={store} client={client} fluxClient={fluxClient}>
      {children}
    </ConsoleTestProvider>
  );
  return { wrapper, store };
};
