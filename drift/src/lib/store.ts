import {
  Action,
  AnyAction,
  configureStore as baseConfigureStore,
  ConfigureStoreOptions as BaseOpts,
  CombinedState,
  Middleware,
  PreloadedState,
  Store,
} from '@reduxjs/toolkit';
import { NoInfer } from '@reduxjs/toolkit/dist/tsHelpers';

import { middleware } from './middleware';
import {
  closeWindow,
  DriftAction,
  setWindowKey,
  setWindowStatus,
  StoreState,
} from './slice';
import { sugarType } from './type';
import { Window } from './window';

// eslint-disable-next-line @typescript-eslint/ban-types
declare type Middlewares<S> = ReadonlyArray<Middleware<{}, S>>;

export interface ConfigureStoreOptions<
  S extends StoreState,
  A extends Action = AnyAction,
  M extends Middlewares<S> = Middlewares<S>
> extends BaseOpts<S, A, M> {
  window: Window<S, A>;
}

export const configureStore = async <
  S extends StoreState,
  A extends Action = AnyAction,
  M extends Middlewares<S> = Middlewares<S>
>({
  window,
  preloadedState,
  ...opts
}: ConfigureStoreOptions<S, A, M>): Promise<Store<S, A | DriftAction>> => {
  let store: Store<S, A | DriftAction> | undefined = undefined;
  preloadedState = await new Promise<
    PreloadedState<CombinedState<NoInfer<S>>> | undefined
  >((resolve) => {
    window.subscribe(({ action, key, state, sendInitialState }) => {
      if (window.isMain() && sendInitialState && store) {
        window.emit({
          state: store.getState() as PreloadedState<CombinedState<NoInfer<S>>>,
          key: window.key(),
        });
      } else if (state && !store) {
        window.ready();
        resolve(state);
      } else if (action && store) {
        store.dispatch({
          ...action,
          type: sugarType(action.type, key),
        });
      }
    });

    if (!window.isMain()) {
      window.emit({ sendInitialState: true, key: window.key() });
    } else {
      resolve(preloadedState);
    }
  });

  store = baseConfigureStore<S, A, M>({
    preloadedState,
    ...opts,
    middleware: (def): M =>
      [...def(), middleware<S, A>(window)] as unknown as M,
  });

  store.dispatch(setWindowKey({ key: window.key() }));
  store.dispatch(setWindowStatus({ key: window.key(), status: 'created' }));

  window.onCloseRequested(() =>
    store?.dispatch(closeWindow({ key: window.key() }))
  );

  return store;
};
