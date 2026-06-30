// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  combineReducers,
  configureStore,
  type EnhancedStore,
  type Reducer,
} from "@reduxjs/toolkit";
import { type Synnax as Client } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Aether, Flux, Pluto, Status, Synnax } from "@synnaxlabs/pluto";
import { aether, eraser, flux, status, synnax } from "@synnaxlabs/pluto/ether";
import {
  render,
  renderHook,
  type RenderHookOptions,
  type RenderHookResult,
  type RenderOptions,
  type RenderResult,
} from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement, useMemo } from "react";
import { Provider } from "react-redux";

import { Cluster } from "@/cluster";
import { Layout } from "@/layout";
import { Project } from "@/project";
import { Session } from "@/session";

const consoleReducer = combineReducers({
  [Layout.SLICE_NAME]: Layout.reducer,
  [Drift.SLICE_NAME]: Drift.reducer,
  [Session.Log.SLICE_NAME]: Session.Log.reducer,
  [Project.SLICE_NAME]: Project.reducer,
  [Cluster.SLICE_NAME]: Cluster.reducer,
  [Session.Nav.SLICE_NAME]: Session.Nav.reducer,
});

export type ConsolePreloadedState = {
  [Layout.SLICE_NAME]?: Layout.SliceState;
  [Session.Log.SLICE_NAME]?: Session.Log.SliceState;
  [Project.SLICE_NAME]?: Project.SliceState;
  [Cluster.SLICE_NAME]?: Cluster.SliceState;
  [Session.Nav.SLICE_NAME]?: Session.Nav.SliceState;
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
    middleware: (getDefault) => getDefault().concat(Session.Nav.MIDDLEWARE),
  });
};

const AETHER_REGISTRY: aether.ComponentRegistry = {
  ...synnax.REGISTRY,
  ...status.REGISTRY,
  ...eraser.REGISTRY,
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
          <Provider store={store}>
            <Session.Modals.Provider>{children}</Session.Modals.Provider>
          </Provider>
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

/**
 * Renders a deep-link resource hook against a minimal Redux store. The store always
 * carries the layout and drift slices that Layout.usePlacer depends on; pass
 * extraReducers for any additional slices the hook reads or writes. Returns the hook's
 * value (the link handler), the Redux store so the spec can assert on placed layouts and
 * dispatched state, and the modal store so the spec can assert on opened modals.
 */
export const renderLinkHook = <H,>(
  useHook: () => H,
  extraReducers: Record<string, Reducer> = {},
): { handler: H; store: EnhancedStore; modals: Session.Modals.Store } => {
  const store = configureStore({
    reducer: combineReducers({
      [Layout.SLICE_NAME]: Layout.reducer,
      [Drift.SLICE_NAME]: Drift.reducer,
      ...extraReducers,
    }),
  });
  const Wrapper = ({ children }: PropsWithChildren) => (
    <Provider store={store}>
      <Session.Modals.Provider>{children}</Session.Modals.Provider>
    </Provider>
  );
  const { result } = renderHook(
    () => ({ handler: useHook(), modals: Session.Modals.useStore("renderLinkHook") }),
    { wrapper: Wrapper },
  );
  return { handler: result.current.handler, store, modals: result.current.modals };
};

export interface RenderHookWithConsoleOptions<Props> extends RenderHookOptions<Props> {
  preloadedState?: ConsolePreloadedState;
  store?: EnhancedStore;
  client?: Client | null;
}

export const renderHookWithConsole = <Result, Props>(
  cb: (props: Props) => Result,
  options: RenderHookWithConsoleOptions<Props> = {},
): RenderHookResult<Result, Props> & { store: EnhancedStore } => {
  const {
    preloadedState,
    store = createTestStore({ preloadedState }),
    client = null,
    ...rest
  } = options;
  const fluxClient = createFluxClient(client);
  const Wrapper = ({ children }: PropsWithChildren) => (
    <ConsoleTestProvider store={store} client={client} fluxClient={fluxClient}>
      {children}
    </ConsoleTestProvider>
  );
  return { ...renderHook(cb, { wrapper: Wrapper, ...rest }), store };
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
