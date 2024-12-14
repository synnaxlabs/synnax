// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Action, UnknownAction } from "@reduxjs/toolkit";

import { type Communicator } from "@/runtime";
import { type StoreState } from "@/state";
import { sugar } from "@/sugar";
import { validateAction } from "@/validate";

/** A store whose state can be retrieved. */
export interface StoreStateGetter<S extends StoreState> {
  getState: () => S;
}

/* A store that can dispatch actions. */
export interface StoreDispatch<A extends Action = UnknownAction> {
  dispatch: (action: A) => void;
}

/**
 * Listens for events from other windows and dispatches them to the store.
 *
 * @param communicator - The runtime to listen for events on.
 * @param getStore - A function returning a store to dispatch events to.
 * @param resolve - A function that resolves a promise requesting
 * the initial store state.
 */
export const listen = async <S extends StoreState, A extends Action = UnknownAction>(
  communicator: Communicator<S, A>,
  getStore: () => (StoreStateGetter<S> & StoreDispatch<A>) | undefined | null,
  resolve: (value: S) => void,
): Promise<void> =>
  await communicator.subscribe(({ action, emitter, state, sendState }) => {
    const s = getStore();
    // case where we're receivign preloaded state.
    if (s == null) {
      if (state != null) return resolve(state);

      return;
    }
    if (action != null) {
      validateAction({ action, emitter });
      return s.dispatch(sugar(action, emitter));
    }
    if (sendState === true && communicator.isMain())
      void communicator.emit({ state: s.getState() }, emitter);
  });
