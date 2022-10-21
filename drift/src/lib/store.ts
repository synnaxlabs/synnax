import {
  Store,
  ConfigureStoreOptions as BaseOpts,
  configureStore as baseConfigureStore,
  Action,
  ThunkMiddleware,
  AnyAction,
} from '@reduxjs/toolkit';
import { sugarType } from './type';
import { middleware } from './middleware';
import { Window, Event } from './window';
import { closeWindow } from './slice';

export interface ConfigureStoreOptions
  extends BaseOpts<
    any,
    Action<any>,
    ThunkMiddleware<any, AnyAction, undefined>[]
  > {
  window: Window;
}

export const configureStore = async ({
  window,
  preloadedState,
  ...opts
}: ConfigureStoreOptions): Promise<Store> => {
  let store: Store | undefined = undefined;
  preloadedState = await new Promise<{ [key: string]: any } | undefined>(
    (resolve) => {
      window.subscribe(({ action, key, state, sendInitialState }: Event) => {
        if (window.isMain() && sendInitialState && store) {
          window.emit({ state: store.getState(), key: window.key() });
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
    }
  );

  store = baseConfigureStore({
    preloadedState,
    ...opts,
    middleware: (def) => [...def(), middleware(window)],
  });

  window.onClose(() => store?.dispatch(closeWindow()));

  return store;
};
