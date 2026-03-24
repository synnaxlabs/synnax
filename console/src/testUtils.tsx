// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers, configureStore, type EnhancedStore } from "@reduxjs/toolkit";
import { Drift } from "@synnaxlabs/drift";
import { Aether, Flux, Pluto, Status, Synnax } from "@synnaxlabs/pluto";
import { aether, flux, status, synnax } from "@synnaxlabs/pluto/ether";
import { createMockWorkers } from "@synnaxlabs/x";
import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, useMemo } from "react";
import { Provider } from "react-redux";

import { Cluster } from "@/cluster";
import { Layout } from "@/layout";
import { Log } from "@/log";
import { Workspace } from "@/workspace";

const consoleReducer = combineReducers({
  [Layout.SLICE_NAME]: Layout.reducer,
  [Drift.SLICE_NAME]: Drift.reducer,
  [Log.SLICE_NAME]: Log.reducer,
  [Workspace.SLICE_NAME]: Workspace.reducer,
  [Cluster.SLICE_NAME]: Cluster.reducer,
});

type ConsolePreloadedState = {
  [Layout.SLICE_NAME]?: Layout.SliceState;
  [Log.SLICE_NAME]?: Log.SliceState;
  [Workspace.SLICE_NAME]?: Workspace.SliceState;
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
    const [w, main] = createMockWorkers();
    aether.render({ comms: w.route("test"), registry: AETHER_REGISTRY });
    return main.route("test") as Aether.ProviderProps["worker"];
  }, []);
  return (
    <Aether.Provider worker={worker} workerKey="test">
      {children}
    </Aether.Provider>
  );
};

const fluxClient = new Flux.Client({
  client: null,
  storeConfig: { ...Pluto.FLUX_STORE_CONFIG },
  handleError: status.createErrorHandler(console.error),
  handleAsyncError: status.createAsyncErrorHandler(console.error),
});

export const ConsoleTestProvider = ({
  store,
  children,
}: PropsWithChildren<{ store: EnhancedStore }>): ReactElement => (
  <AetherTestProvider>
    <Status.Aggregator>
      <Synnax.TestProvider client={null}>
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
    <ConsoleTestProvider store={store}>{children}</ConsoleTestProvider>
  );
  return { ...render(ui, { wrapper: Wrapper, ...rest }), store };
};
