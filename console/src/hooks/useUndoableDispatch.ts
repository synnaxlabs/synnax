// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type Selector, type UnknownAction } from "@reduxjs/toolkit";
import { useDebouncedCallback } from "@synnaxlabs/pluto";
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
  stateUpdateActionCreator: (state: SelectedType) => UnknownAction,
  waitFor: number = 0,
  size: number = 20,
) => {
  const store = useStore<StoreType>();

  const history = useRef<History<SelectedType>>({
    past: [],
    present: null,
    future: [],
  });

  const updateState = useDebouncedCallback(
    () => {
      const currentState = selector(store.getState());
      if (deep.equal(history.current.present, currentState)) return;
      if (history.current.present != null)
        history.current.past.push(history.current.present);
      if (history.current.past.length > size) history.current.past.shift();
      history.current.present = currentState;
      history.current.future = [];
    },
    waitFor,
    [selector, store, history, size],
  );

  const undoableDispatch = useCallback(
    (action: UnknownAction) => {
      history.current.present ??= selector(store.getState());
      updateState();
      store.dispatch(action);
    },
    [history, selector, store, updateState],
  );

  const undo = useCallback(() => {
    const lastState = history.current.past.pop();
    if (lastState == null) return;
    if (history.current.present != null)
      history.current.future.unshift(history.current.present);
    history.current.present = lastState;
    store.dispatch(stateUpdateActionCreator(lastState));
  }, [history, store, stateUpdateActionCreator]);

  const redo = useCallback(() => {
    const nextState = history.current.future.shift();
    if (nextState == null) return;
    if (history.current.present != null)
      history.current.past.push(history.current.present);
    history.current.present = nextState;
    store.dispatch(stateUpdateActionCreator(nextState));
  }, [history, store, stateUpdateActionCreator]);

  return [undoableDispatch as Dispatch, undo, redo] as const;
};
