import { Action, AnyAction, Store } from '@reduxjs/toolkit';

import { Runtime } from './runtime';
import { DriftAction, PreloadedState, StoreState } from './state';
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
  resolve: (value: PreloadedState<S>) => void
) => {
  runtime.subscribe(({ action, emitter, state, sendState }) => {
    if (!store) return state && resolve(state);
    if (action) return store.dispatch(sugar(action, emitter));
    if (sendState && runtime.isMain())
      runtime.emit({ state: store.getState() as PreloadedState<S> }, emitter);
  });
};
