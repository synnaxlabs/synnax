import {
  Action,
  AnyAction,
  configureStore as base,
  ConfigureStoreOptions as BaseOpts,
  Store,
} from '@reduxjs/toolkit';

import { listen } from './listener';
import { configureMiddleware, Middlewares } from './middleware';
import { Runtime } from './runtime';
import {
  closeWindow,
  DriftAction,
  PreloadedState,
  setWindowKey,
  setWindowStatus,
  StoreState,
} from './state';
import { MAIN_WINDOW } from './window';

/**
 * Extends the default configureStore options to add a runtime argument.
 * See configureStore for more details.
 */
export interface ConfigureStoreOptions<
  S extends StoreState,
  A extends Action = AnyAction,
  M extends Middlewares<S> = Middlewares<S>
> extends BaseOpts<S, A, M> {
  runtime: Runtime<S, A>;
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
  let store: Store<S, A | DriftAction> | undefined = undefined;

  preloadedState = await new Promise<PreloadedState<S> | undefined>(
    (resolve) => {
      listen(runtime, store, resolve);
      if (runtime.isMain()) resolve(preloadedState);
      else runtime.emit({ sendState: true }, MAIN_WINDOW);
    }
  );

  store = base<S, A, M>({
    ...opts,
    preloadedState,
    middleware: configureMiddleware(middleware, runtime),
  });

  store.dispatch(setWindowKey(runtime.key()));
  store.dispatch(setWindowStatus('created'));
  runtime.onCloseRequested(() => store?.dispatch(closeWindow()));
  runtime.ready();

  return store;
};
