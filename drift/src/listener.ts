import {
  Action,
  AnyAction,
  CombinedState,
  PreloadedState,
  Store,
} from '@reduxjs/toolkit';
import { NoInfer } from '@reduxjs/toolkit/dist/tsHelpers';

import { Runtime } from './runtime';
import { DriftAction, StoreState } from './state';
import { sugar } from './sugar';

/**
 * Listens for events from other windows and dispatches them to the store.
 *
 * @param runtime - The runtime to listen for events on.
 * @param store - The store to dispatch events to.
 * @param resolve - A function that resolves a promise requesting
 * the initial store state.
 */
export const listen = <S extends StoreState, A extends Action = AnyAction>(
  runtime: Runtime<S, A>,
  store: Store<S, A | DriftAction> | undefined,
  resolve: (value: PreloadedState<CombinedState<NoInfer<S>>>) => void
) => {
  runtime.subscribe(({ action, emitter, state, sendInitialState }) => {
    if (!store) {
      if (state) resolve(state);
      return;
    }

    if (action) {
      store.dispatch(sugar(action, emitter));
    } else if (sendInitialState && runtime.isMain()) {
      runtime.emit({
        state: store.getState() as PreloadedState<CombinedState<NoInfer<S>>>,
      });
    }
  });
};
