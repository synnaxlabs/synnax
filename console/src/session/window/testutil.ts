// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Action,
  combineReducers,
  configureStore,
  type Middleware,
  type Reducer,
} from "@reduxjs/toolkit";
import { Drift, MAIN_WINDOW } from "@synnaxlabs/drift";

import { type Documents } from "@/session/window/keyed";

/**
 * The documents map for one window, defaulting to the main window a spec dispatches
 * from.
 */
export const inWindow = <Doc>(
  documents: Documents<Doc>,
  windowKey: string = MAIN_WINDOW,
): Record<string, Documents<Doc>> => ({ [windowKey]: documents });

/** One window's view of a document, for specs asserting on window-keyed slices. */
export const documentIn = <Doc>(
  slice: { windows: Record<string, Documents<Doc>> },
  key: string,
  windowKey: string = MAIN_WINDOW,
): Doc | undefined => slice.windows[windowKey]?.[key];

/** Drift state naming the main window, for specs building a window-keyed store state. */
export const DRIFT_STATE: Drift.StoreState = {
  [Drift.SLICE_NAME]: Drift.ZERO_SLICE_STATE,
};

/**
 * Drift state naming the main window and one torn-off window, for specs that open or
 * close a second window.
 * @param key - The window's key, which window-keyed slices store their state under.
 * @param label - The runtime label drift addresses the window by.
 */
export const createDriftStateWithWindow = (
  key: string,
  label: string,
): Drift.SliceState => {
  const zero = Drift.ZERO_SLICE_STATE;
  return {
    ...zero,
    windows: {
      ...zero.windows,
      [label]: {
        key,
        stage: "created",
        processCount: 0,
        reserved: true,
        focusCount: 0,
        centerCount: 0,
        ordinal: 2,
      },
    },
    labelKeys: { ...zero.labelKeys, [label]: key },
    keyLabels: { ...zero.keyLabels, [key]: label },
    nextOrdinal: 3,
  };
};

export interface SliceStoreParams<Name extends string, S> {
  name: Name;
  reducer: Reducer<S>;
  preloadedState: S;
  /** The slice's key-injecting middleware. Window-scoped actions need it. */
  middleware?: Middleware[];
}

/**
 * A store holding one window-keyed slice plus the drift slice its selectors read the
 * current window from, running the slice's own middleware.
 */
export const createSliceStore = <Name extends string, S, A extends Action = Action>({
  name,
  reducer,
  preloadedState,
  middleware = [],
}: SliceStoreParams<Name, S>) =>
  configureStore<Record<Name, S> & Drift.StoreState, A>({
    reducer: combineReducers({
      [name]: reducer,
      [Drift.SLICE_NAME]: Drift.reducer,
    }) as unknown as Reducer<Record<Name, S> & Drift.StoreState, A>,
    preloadedState: {
      [name]: preloadedState,
      ...DRIFT_STATE,
    } as Record<Name, S> & Drift.StoreState,
    middleware: (getDefault) =>
      getDefault({ serializableCheck: false }).concat(middleware) as never,
  });
