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
import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";

import { Layout } from "@/layout";
import { Log } from "@/log";

const consoleReducer = combineReducers({
  [Layout.SLICE_NAME]: Layout.reducer,
  [Drift.SLICE_NAME]: Drift.reducer,
  [Log.SLICE_NAME]: Log.reducer,
});

type ConsolePreloadedState = {
  [Layout.SLICE_NAME]?: Layout.SliceState;
  [Log.SLICE_NAME]?: Log.SliceState;
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

export const ConsoleTestProvider = ({
  store,
  children,
}: PropsWithChildren<{ store: EnhancedStore }>): ReactElement => (
  <Provider store={store}>{children}</Provider>
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
