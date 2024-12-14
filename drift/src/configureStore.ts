// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Action as CoreAction,
  configureStore as base,
  type ConfigureStoreOptions as BaseOpts,
  type EnhancedStore,
  type StoreEnhancer,
  type Tuple,
  type UnknownAction,
} from "@reduxjs/toolkit";

import { listen } from "@/listener";
import { configureMiddleware, type Middlewares } from "@/middleware";
import { type Runtime } from "@/runtime";
import {
  type Action,
  closeWindow,
  internalSetInitial,
  setWindowStage,
  SLICE_NAME,
  type StoreState,
} from "@/state";
import { syncInitial } from "@/sync";
import { MAIN_WINDOW, type WindowProps } from "@/window";

export type Enhancers = readonly StoreEnhancer[];

/**
 * Extends the default configureStore options to add a runtime argument.
 * See configureStore for more details.
 */
export interface ConfigureStoreOptions<
  S extends StoreState,
  A extends CoreAction = UnknownAction,
  M extends Tuple<Middlewares<S>> = Tuple<Middlewares<S>>,
  E extends Tuple<Enhancers> = Tuple<Enhancers>,
> extends Omit<BaseOpts<S, A, M, E>, "preloadedState"> {
  runtime: Runtime<S, A | Action>;
  debug?: boolean;
  preloadedState?: S | (() => Promise<S | undefined>);
  enablePrerender?: boolean;
  defaultWindowProps?: Omit<WindowProps, "key">;
}

/* The internal function. We export with a strict type annotation so TS doesn't complain */
const configureStoreInternal = async <
  S extends StoreState,
  A extends CoreAction = UnknownAction,
  M extends Tuple<Middlewares<S>> = Tuple<Middlewares<S>>,
  E extends Tuple<Enhancers> = Tuple<Enhancers>,
>({
  runtime,
  preloadedState,
  middleware,
  debug = false,
  enablePrerender = true,
  defaultWindowProps,
  ...opts
}: ConfigureStoreOptions<S, A, M, E>): Promise<EnhancedStore<S, A | Action>> => {
  await runtime.configure();

  let store: EnhancedStore<S, A | Action> | undefined;
  // eslint-disable-next-line prefer-const
  store = base<S, A, M, E>({
    ...opts,
    preloadedState: await receivePreloadedState(
      debug,
      runtime,
      () => store,
      defaultWindowProps,
      preloadedState,
    ),
    middleware: configureMiddleware(middleware, runtime, debug),
  });

  await syncInitial(store.getState().drift, store.dispatch, runtime, debug);
  const label = runtime.label();
  store.dispatch(
    internalSetInitial({ enablePrerender, defaultWindowProps, debug, label }),
  );
  console.log("Setting created");
  store.dispatch(setWindowStage({ stage: "created" }));
  runtime.onCloseRequested(() => store?.dispatch(closeWindow({})));
  return store;
};

const receivePreloadedState = async <
  S extends StoreState,
  A extends CoreAction = UnknownAction,
>(
  debug: boolean,
  runtime: Runtime<S, A>,
  store: () => EnhancedStore<S, A | Action> | undefined,
  defaultWindowProps: Omit<WindowProps, "key"> | undefined,
  preloadedState: (() => Promise<S | undefined>) | S | undefined,
): Promise<S | undefined> =>
  await new Promise<S | undefined>((resolve) => {
    void (async () => {
      await listen(runtime, store, resolve);
      if (runtime.isMain())
        if (typeof preloadedState === "function")
          preloadedState()
            .then((s) => resolve(resetInitialState<S>(defaultWindowProps, debug, s)))
            .catch(console.error);
        else resolve(resetInitialState<S>(defaultWindowProps, debug, preloadedState));
      else await runtime.emit({ sendState: true }, MAIN_WINDOW);
    })();
  });

/**
 * configureStore replaces the standard Redux Toolkit configureStore function
 * with one that enables drift to synchronize state between windows. The API
 * is identical to the standard configureStore function, except for two
 * important differences.
 *
 * @param options.runtime - The core runtime of the application. This should
 * be chosen based on the platform you are running on (Tauri, Electron, etc.).
 * @param options.debug - If true, drift will log debug information to the
 * console. @default false
 * @param props.enablePrerender - If true, drift will create an invisible, pre-rendered
 * window before it is needed. While it adds an additional process to your application,
 * it also dramatically reduces the time it takes to open a new window. @default true
 * @param props.defaultWindowProps - A partial set of window props to merge with
 * the props passed to drift.createWindow. This is useful for setting default window
 * properties, especially with prerendering. @default {}
 * @param options - The standard Redux Toolkit configureStore options.
 *
 * @returns A !PROMISE! that resolves to a Redux store. This is necessary because
 * the store must receive it's initial state from the main window, which is
 * an asynchronous operation. The promise will resolve when the store is configured
 * and the window is ready for use.
 */
export const configureStore: <
  S extends StoreState,
  A extends CoreAction = UnknownAction,
  M extends Tuple<Middlewares<S>> = Tuple<Middlewares<S>>,
  E extends Tuple<Enhancers> = Tuple<Enhancers>,
>(
  options: ConfigureStoreOptions<S, A, M, E>,
) => Promise<EnhancedStore<S, A | Action>> = configureStoreInternal;

export const resetInitialState = <S extends StoreState>(
  defaultWindowProps?: Omit<WindowProps, "key">,
  debug?: boolean,
  state?: S,
): S | undefined => {
  if (state == null) return state;
  const drift = state[SLICE_NAME];
  drift.config.debug = debug ?? drift.config.debug;
  drift.windows = Object.fromEntries(
    Object.entries(drift.windows)
      .filter(([, window]) => window.reserved)
      .map(([key, window]) => {
        if (defaultWindowProps?.visible != null)
          window.visible = defaultWindowProps.visible;
        window.focusCount = 0;
        window.centerCount = 0;
        window.processCount = 0;
        return [key, window];
      }),
  );
  return state;
};
