// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type {
  Action,
  AnyAction,
  ConfigureStoreOptions as BaseOpts,
  EnhancedStore,
  StoreEnhancer,
} from "@reduxjs/toolkit";
import { configureStore as base } from "@reduxjs/toolkit";

import { listen } from "./listener";
import { Middlewares, configureMiddleware } from "./middleware";
import { Runtime } from "./runtime";
import {
  DriftAction,
  PreloadedState,
  StoreState,
  setWindowLabel,
  setWindowStage,
  closeWindow,
  setPrererenderEnabled,
} from "./state";
import { syncInitial } from "./sync";
import { MAIN_WINDOW } from "./window";

export type Enhancers = readonly StoreEnhancer[];

/**
 * Extends the default configureStore options to add a runtime argument.
 * See configureStore for more details.
 */
export interface ConfigureStoreOptions<
  S extends StoreState,
  A extends Action = AnyAction,
  M extends Middlewares<S> = Middlewares<S>,
  E extends Enhancers = [StoreEnhancer]
> extends Omit<BaseOpts<S, A, M, E>, "preloadedState"> {
  runtime: Runtime<S, A>;
  debug?: boolean;
  preloadedState?: PreloadedState<S> | (() => Promise<PreloadedState<S> | undefined>);
  enablePrerender?: boolean;
}

/**
 * configureStore replaces the standard Redux Toolkit configureStore function
 * with one that enables drift to synchronize state between windows. The API
 * is identical to the standard configureStore function, except for two
 * important differences.
 *
 * @param options.runtime - The core runtime of the application. This should
 * be chosen based on the platform you are running on (Tauri, Electron, etc.).
 * @param options - The standard Redux Toolkit configureStore options.
 *
 * @returns A !PROMISE! that resolves to a Redux store. This is necessary because
 * the store must receive it's initial state from the main window, which is
 * an asynchronous operation. The promise will resolve when the store is configured
 * and the window is ready for use.
 */
export const configureStore = async <
  S extends StoreState,
  A extends Action = AnyAction,
  M extends Middlewares<S> = Middlewares<S>,
  E extends Enhancers = [StoreEnhancer]
>({
  runtime,
  preloadedState,
  middleware,
  debug = false,
  enablePrerender = true,
  ...opts
}: ConfigureStoreOptions<S, A, M, E>): Promise<EnhancedStore<S, A | DriftAction>> => {
  // eslint-disable-next-line prefer-const
  let store: EnhancedStore<S, A | DriftAction> | undefined;
  // eslint-disable-next-line prefer-const
  store = base<S, A, M, E>({
    ...opts,
    preloadedState: await receivePreloadedState(runtime, () => store, preloadedState),
    middleware: configureMiddleware(middleware, runtime, debug),
  });

  store.dispatch(setPrererenderEnabled({ value: enablePrerender }));
  await syncInitial(store.getState().drift, runtime, debug);
  store.dispatch(setWindowLabel({ label: runtime.label() }));
  store.dispatch(setWindowStage({ stage: "created" }));
  runtime.onCloseRequested(() => store?.dispatch(closeWindow({})));
  if (runtime.isMain()) void runtime.setVisible(true);

  return store;
};

const receivePreloadedState = async <
  S extends StoreState,
  A extends Action = AnyAction
>(
  runtime: Runtime<S, A>,
  store: () => EnhancedStore<S, A | DriftAction> | undefined,
  preloadedState:
    | (() => Promise<PreloadedState<S> | undefined>)
    | PreloadedState<S>
    | undefined
): Promise<PreloadedState<S> | undefined> =>
  await new Promise<PreloadedState<S> | undefined>((resolve) => {
    void listen(runtime, store, resolve);
    if (runtime.isMain()) {
      if (typeof preloadedState === "function")
        preloadedState().then(resolve).catch(console.error);
      else resolve(preloadedState);
    } else void runtime.emit({ sendState: true }, MAIN_WINDOW);
  });
