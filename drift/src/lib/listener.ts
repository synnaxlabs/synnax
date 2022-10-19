import {
  Store,
  ConfigureStoreOptions as BaseOpts,
  configureStore as baseConfigureStore,
  Action,
  ThunkMiddleware,
  AnyAction,
} from '@reduxjs/toolkit';
import { embedDriftMD } from './actions';
import { middleware } from './middleware';
import { Runtime, Event } from './runtime';

export interface ConfigureStoreOptions
  extends BaseOpts<
    any,
    Action<any>,
    ThunkMiddleware<any, AnyAction, undefined>[]
  > {
  runtime: Runtime;
}

export const configureStore = async ({
  runtime,
  preloadedState,
  ...opts
}: ConfigureStoreOptions): Promise<Store> => {
  let store: Store | undefined = undefined;
  preloadedState = await new Promise<{ [key: string]: any }>((resolve) => {
    runtime.subscribe(({ action, winKey, state, sendInitialState }: Event) => {
      if (runtime.isMain() && sendInitialState && store) {
        runtime.emit({ state: store.getState(), winKey: runtime.winKey() });
      } else if (state && !store) {
        runtime.ready();
        resolve(state);
      } else if (action && store) {
        store.dispatch({
          ...action,
          type: embedDriftMD({ type: action.type, winKey }),
        });
      }
    });
    if (!runtime.isMain()) {
      runtime.emit({ sendInitialState: true, winKey: runtime.winKey() });
    } else {
      resolve({});
    }
  });
  store = baseConfigureStore({
    preloadedState,
    ...opts,
    middleware: (getDefaultMiddleware) => [
      ...getDefaultMiddleware(),
      middleware({ runtime }),
    ],
  });
  return store;
};
