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
import { eraser } from "@synnaxlabs/pluto/ether";
import {
  render,
  renderHook,
  type RenderHookOptions,
  type RenderHookResult,
  type RenderOptions,
  type RenderResult,
} from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";

import { Session } from "@/session";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil/Synnax";

export type ConsolePreloadedState = Partial<Session.State>;

export interface ConsoleTestProviderOptions {
  preloadedState?: ConsolePreloadedState;
}

export const createTestStore = (options: ConsoleTestProviderOptions = {}) => {
  const { preloadedState } = options;
  return configureStore({
    reducer: Session.reducer,
    // combineReducers fills any slice omitted from the partial preloaded state.
    preloadedState: preloadedState as Session.State | undefined,
    middleware: (getDefault) => getDefault().concat(...Session.BASE_MIDDLEWARE),
  });
};

/** The console test store, typed with the full production root state. */
export type TestStore = ReturnType<typeof createTestStore>;

// The eraser aether component is required by console widgets but is not part of pluto's
// default test registry, so it is injected via additionalRegistry.
const ADDITIONAL_REGISTRY = eraser.REGISTRY;

const composeConsole = (
  SynnaxWrapper: FC<PropsWithChildren>,
  store: EnhancedStore,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <SynnaxWrapper>
      <Provider store={store}>
        <Session.Modals.Provider>{children}</Session.Modals.Provider>
      </Provider>
    </SynnaxWrapper>
  );
  return Wrapper;
};

export interface RenderWithConsoleOptions extends RenderOptions {
  preloadedState?: ConsolePreloadedState;
  store?: TestStore;
}

export const renderWithConsole = (
  ui: ReactElement,
  options: RenderWithConsoleOptions = {},
): RenderResult & { store: TestStore } => {
  const {
    preloadedState,
    store = createTestStore({ preloadedState }),
    ...rest
  } = options;
  const Wrapper = composeConsole(
    createSynnaxWrapper({ client: null, additionalRegistry: ADDITIONAL_REGISTRY }),
    store,
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
      [Session.Layout.SLICE_NAME]: Session.Layout.reducer,
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
  store?: TestStore;
  client?: Client | null;
}

export const renderHookWithConsole = <Result, Props>(
  cb: (props: Props) => Result,
  options: RenderHookWithConsoleOptions<Props> = {},
): RenderHookResult<Result, Props> & { store: TestStore } => {
  const {
    preloadedState,
    store = createTestStore({ preloadedState }),
    client = null,
    ...rest
  } = options;
  const Wrapper = composeConsole(
    createSynnaxWrapper({ client, additionalRegistry: ADDITIONAL_REGISTRY }),
    store,
  );
  return { ...renderHook(cb, { wrapper: Wrapper, ...rest }), store };
};

export interface CreateConsoleWrapperArgs {
  client: Client | null;
  preloadedState?: ConsolePreloadedState;
  store?: TestStore;
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
  store: TestStore;
}> => {
  const SynnaxWrapper = await createAsyncSynnaxWrapper({
    client,
    additionalRegistry: ADDITIONAL_REGISTRY,
  });
  return { wrapper: composeConsole(SynnaxWrapper, store), store };
};
