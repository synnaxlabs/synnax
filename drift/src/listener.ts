import { Action, AnyAction, Store } from "@reduxjs/toolkit";

import { Communicator } from "./runtime";
import { DriftAction, PreloadedState, StoreState } from "./state";
import { sugar } from "./sugar";

interface StoreStateGetter<S extends StoreState> {
	getState(): S;
}

interface StoreDispatch<A extends Action = AnyAction> {
	dispatch(action: A): void;
}

/**
 * Listens for events from other windows and dispatches them to the store.
 *
 * @param communicator - The runtime to listen for events on.
 * @param store - The store to dispatch events to.
 * @param resolve - A function that resolves a promise requesting
 * the initial store state.
 */
export const listen = <S extends StoreState, A extends Action = AnyAction>(
	communicator: Communicator<S, A>,
	store: () => (StoreStateGetter<S> & StoreDispatch<A>) | undefined,
	resolve: (value: PreloadedState<S>) => void
) => {
	communicator.subscribe(({ action, emitter, state, sendState }) => {
		const s = store();
		if (!s) return state && resolve(state);
		if (action) return s.dispatch(sugar(action, emitter));
		if (sendState && communicator.isMain())
			communicator.emit({ state: s.getState() as PreloadedState<S> }, emitter);
	});
};
