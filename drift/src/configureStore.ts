// Copyright 2022 Synnax Labs, Inc.
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
  Store,
} from "@reduxjs/toolkit";
import { configureStore as base } from "@reduxjs/toolkit";

import { listen } from "./listener";
import { Middlewares, configureMiddleware } from "./middleware";
import { Runtime } from "./runtime";
import {
  DriftAction,
  PreloadedState,
  StoreState,
  closeWindow,
  setWindowKey,
  setWindowState,
} from "./state";
import { MAIN_WINDOW } from "./window";

/**
 * Extends the default configureStore options to add a runtime argument.
 * See configureStore for more details.
 */
export interface ConfigureStoreOptions<
  S extends StoreState,
  A extends Action = AnyAction,
  M extends Middlewares<S> = Middlewares<S>
> extends Omit<BaseOpts<S, A, M>, "preloadedState"> {
  runtime: Runtime<S, A>;
  preloadedState: PreloadedState<S> | (() => Promise<PreloadedState<S>>);
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
  M extends Middlewares<S> = Middlewares<S>
>({
  runtime,
  preloadedState,
  middleware,
  ...opts
}: ConfigureStoreOptions<S, A, M>): Promise<Store<S, A | DriftAction>> => {
  // eslint-disable-next-line prefer-const
  let store: Store<S, A | DriftAction> | undefined;
  // eslint-disable-next-line prefer-const
  store = base<S, A, M>({
    ...opts,
    preloadedState: await receivePreloadedState(runtime, () => store, preloadedState),
    middleware: configureMiddleware(middleware, runtime),
  });

  store.dispatch(setWindowKey(runtime.key()));
  store.dispatch(setWindowState("created"));
  runtime.onCloseRequested(() => store?.dispatch(closeWindow()));
  runtime.ready();

  return store;
};

const receivePreloadedState = async <
  S extends StoreState,
  A extends Action = AnyAction
>(
  runtime: Runtime<S, A>,
  store: () => Store<S, A | DriftAction> | undefined,
  preloadedState: (() => Promise<PreloadedState<S>>) | PreloadedState<S> | undefined
): Promise<PreloadedState<S> | undefined> => {
  return await new Promise<PreloadedState<S> | undefined>((resolve) => {
    listen(runtime, store, resolve);
    if (runtime.isMain()) {
      if (typeof preloadedState === "function")
        preloadedState().then(resolve).catch(console.error);
      else resolve(preloadedState);
    } else runtime.emit({ sendState: true }, MAIN_WINDOW);
  });
};
