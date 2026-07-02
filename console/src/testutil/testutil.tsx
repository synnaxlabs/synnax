// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type EnhancedStore } from "@reduxjs/toolkit";
import { type Synnax as Client } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { eraser } from "@synnaxlabs/pluto/ether";
import { deep } from "@synnaxlabs/x";
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

/**
 * Builds the console test store the same way the app builds its store: the full root
 * reducer wired through drift's middleware, but with a NoopRuntime in place of the Tauri
 * runtime. Window actions (Drift.focusWindow, createWindow) resolve their labels exactly
 * as they do in production. Async because Drift.configureStore awaits runtime setup. This
 * is the one console store — every render helper below is backed by it.
 */
export const createTestStore = async (options: ConsoleTestProviderOptions = {}) => {
  const { preloadedState } = options;
  return await Drift.configureStore({
    runtime: new Drift.NoopRuntime(),
    reducer: Session.reducer,
    preloadedState:
      preloadedState != null
        ? deep.copy({ ...Session.ZERO_STATE, ...preloadedState })
        : undefined,
    // The layout middleware is omitted: it drives real Tauri window creation, which
    // cannot run in jsdom. Everything else matches the production store.
    middleware: (getDefault) => getDefault().concat(...Session.Nav.MIDDLEWARE),
    enablePrerender: false,
  });
};

export type TestStore = Awaited<ReturnType<typeof createTestStore>>;

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

export const renderWithConsole = async (
  ui: ReactElement,
  options: RenderWithConsoleOptions = {},
): Promise<RenderResult & { store: TestStore }> => {
  const { preloadedState, store, ...rest } = options;
  const resolvedStore = store ?? (await createTestStore({ preloadedState }));
  const Wrapper = composeConsole(
    createSynnaxWrapper({ client: null, additionalRegistry: ADDITIONAL_REGISTRY }),
    resolvedStore,
  );
  return { ...render(ui, { wrapper: Wrapper, ...rest }), store: resolvedStore };
};

export const renderLinkHook = async <H,>(
  useHook: () => H,
): Promise<{ handler: H; store: TestStore; modals: Session.Modals.Store }> => {
  const store = await createTestStore();
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

export const renderHookWithConsole = async <Result, Props>(
  cb: (props: Props) => Result,
  options: RenderHookWithConsoleOptions<Props> = {},
): Promise<RenderHookResult<Result, Props> & { store: TestStore }> => {
  const { preloadedState, store, client = null, ...rest } = options;
  const resolvedStore = store ?? (await createTestStore({ preloadedState }));
  const Wrapper = composeConsole(
    createSynnaxWrapper({ client, additionalRegistry: ADDITIONAL_REGISTRY }),
    resolvedStore,
  );
  return { ...renderHook(cb, { wrapper: Wrapper, ...rest }), store: resolvedStore };
};

export interface CreateConsoleWrapperArgs {
  client: Client | null;
  preloadedState?: ConsolePreloadedState;
  store?: TestStore;
}

export const createConsoleWrapper = async ({
  client,
  preloadedState,
  store,
}: CreateConsoleWrapperArgs): Promise<{
  wrapper: FC<PropsWithChildren>;
  store: TestStore;
}> => {
  const resolvedStore = store ?? (await createTestStore({ preloadedState }));
  const SynnaxWrapper = await createAsyncSynnaxWrapper({
    client,
    additionalRegistry: ADDITIONAL_REGISTRY,
  });
  return {
    wrapper: composeConsole(SynnaxWrapper, resolvedStore),
    store: resolvedStore,
  };
};
