// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ActionReducerMapBuilder,
  createAction,
  type Middleware,
  type PayloadAction,
} from "@reduxjs/toolkit";
import { Drift, MAIN_WINDOW } from "@synnaxlabs/drift";
import { type record } from "@synnaxlabs/x";

export interface RemovedPayload {
  windowKey: string;
}

/**
 * Announces that a window is gone, so the slices keying state by window can drop it.
 * Drift's own close removes only its bookkeeping; without this every window ever opened
 * would leave an entry behind, and window keys are minted fresh per open.
 */
export const removed = createAction<RemovedPayload>("window/removed");

interface WindowedState {
  windows: Record<string, unknown>;
}

/** Drops the closed window's entry. For a slice's extraReducers. */
export const handleRemoved = <S extends WindowedState>(
  builder: ActionReducerMapBuilder<S>,
): void => {
  builder.addCase(
    removed,
    (state, { payload: { windowKey } }: PayloadAction<RemovedPayload>) => {
      delete state.windows[windowKey];
    },
  );
};

const windowKeys = (state: Drift.StoreState): string[] =>
  Object.values(state[Drift.SLICE_NAME].labelKeys);

/**
 * Announces every window the action actually removed. A close asks, it does not tell:
 * Drift defers one while the window has a process registered, and the process
 * completing is what finally lands it. Watching the windows themselves rather than the
 * close means a deferred window keeps its state for as long as it is open.
 */
export const removalMiddleware: Middleware<record.Unknown> =
  (store) => (next) => (action) => {
    if (!Drift.closeWindow.match(action) && !Drift.completeProcess.match(action))
      return next(action);
    const before = windowKeys(store.getState() as Drift.StoreState);
    const result = next(action);
    const after = new Set(windowKeys(store.getState() as Drift.StoreState));
    before
      // Main's key recurs across launches, so the close that ends a session must
      // not drop the state the next one reuses.
      .filter((key) => !after.has(key) && key !== MAIN_WINDOW)
      .forEach((windowKey) => store.dispatch(removed({ windowKey })));
    return result;
  };
