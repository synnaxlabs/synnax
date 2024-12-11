// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type Selector, type UnknownAction } from "@reduxjs/toolkit";
import { deep } from "@synnaxlabs/x";
import { useCallback, useRef } from "react";
import { useStore } from "react-redux";

interface History<T> {
  past: T[];
  present: T | null;
  future: T[];
}

export const useUndoableDispatch = <
  StoreType extends {} = {},
  SelectedType = StoreType,
>(
  selector: Selector<StoreType, SelectedType>,
  actionCreator: (state: SelectedType) => UnknownAction,
  waitFor: number = 0,
) => {
  const store = useStore<StoreType>();

  const history = useRef<History<SelectedType>>({
    past: [],
    present: null,
    future: [],
  });

  const updateState = useCallback(() => {
    const currentState = selector(store.getState());
    if (deep.equal(history.current.present, currentState)) return;
    if (history.current.present != null)
      history.current.past.push(history.current.present);
    history.current.present = currentState;
    history.current.future = [];
  }, [store, selector]);

  let timeoutID: NodeJS.Timeout;

  const undoableDispatch = useCallback(
    (action: UnknownAction) => {
      history.current.present ??= selector(store.getState());

      if (waitFor > 0) {
        clearTimeout(timeoutID);
        timeoutID = setTimeout(updateState, waitFor);
      } else updateState();

      store.dispatch(action);
    },
    [store, selector, waitFor, updateState],
  );

  const undo = useCallback(() => {
    const lastState = history.current.past.pop();
    if (lastState == null) return;
    if (history.current.present != null)
      history.current.future.unshift(history.current.present);
    history.current.present = lastState;
    store.dispatch(actionCreator(lastState));
  }, [store, actionCreator]);

  const redo = useCallback(() => {
    const nextState = history.current.future.shift();
    if (nextState == null) return;
    if (history.current.present != null)
      history.current.past.push(history.current.present);
    history.current.present = nextState;
    store.dispatch(actionCreator(nextState));
  }, [store, actionCreator]);

  return [undoableDispatch as Dispatch, undo, redo] as const;
};
