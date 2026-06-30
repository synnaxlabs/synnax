// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type {
  Dispatch,
  Middleware,
  MiddlewareAPI,
  PayloadAction,
} from "@reduxjs/toolkit";
import { Drift, MAIN_WINDOW, selectWindowKey } from "@synnaxlabs/drift";
import { Mosaic } from "@synnaxlabs/pluto";
import { runtime } from "@synnaxlabs/x";

import { select, selectSliceState } from "@/layered/service/selectors";
import {
  clearProject,
  MOSAIC_WINDOW_TYPE,
  moveMosaicTab,
  type MoveMosaicTabPayload,
  place,
  type PlacePayload,
  remove,
  type RemovePayload,
  setProject,
  type SetProjectPayload,
  type StoreState,
  type WindowProps,
} from "@/layered/service/slice";

export interface MiddlewareEffectArgs<S, LP, DP> {
  store: MiddlewareAPI<Dispatch<{ payload: LP | DP; type: string }>, S>;
  action: PayloadAction<LP>;
}

export type MiddlewareEffect<S, LP, DP = LP> = (
  args: MiddlewareEffectArgs<S, LP, DP>,
) => void;

export const effectMiddleware =
  <S, LP, DP>(
    deps: string[],
    effects: Array<MiddlewareEffect<S, LP | DP, DP | LP>>,
    before: boolean = false,
  ): Middleware<Dispatch<PayloadAction<LP>>, S, Dispatch<PayloadAction<LP>>> =>
  (store) =>
  (next) =>
  (unknownAction) => {
    const action = unknownAction as PayloadAction<LP>;
    if (before && deps.includes(action.type))
      effects.forEach((factory) => factory({ store, action }));
    const state = next(unknownAction);
    if (!before && deps.includes(action.type))
      effects.forEach((factory) => factory({ store, action }));
    return state;
  };

export const closeWindowOnEmptyMosaicEffect: MiddlewareEffect<
  StoreState & Drift.StoreState,
  MoveMosaicTabPayload | RemovePayload,
  Drift.CloseWindowPayload
> = ({ store }) => {
  const s = store.getState();
  if (selectWindowKey(s) !== MAIN_WINDOW) return;
  const { mosaics } = selectSliceState(s);
  // Close windows with empty mosaics.
  Object.entries(mosaics).forEach(([k, { root }]) => {
    if (k === Drift.MAIN_WINDOW || !Mosaic.isEmpty(root)) return;
    const win = Drift.selectWindow(s, k);
    if (win != null) store.dispatch(Drift.closeWindow({ key: k }));
  });
  // Close windows whose mosaics no longer exist.
  const windows = Drift.selectWindows(s);
  windows.forEach((win) => {
    if (win.key == null) {
      console.error("UNEXPECTED ERROR: Drift WindowState.key is null");
      return;
    }
    if (
      win.key == null ||
      !win.key.startsWith(MOSAIC_WINDOW_TYPE) ||
      win.key in mosaics
    )
      return;
    store.dispatch(Drift.closeWindow({ key: win.key }));
  });
};

const createWindowAction = (
  key: string,
  name: string,
  props?: WindowProps,
): ReturnType<typeof Drift.createWindow> => {
  // Purge props that are not explicitly needed within the drift window state.
  const purgedProps = {
    ...props,
    navTop: undefined,
    showTitle: undefined,
    decorations: runtime.getOS() !== "Windows",
  };
  return Drift.createWindow({
    ...purgedProps,
    url: "/",
    key,
    title: name,
  });
};
export const createWindowOnPlaceEffect: MiddlewareEffect<
  StoreState & Drift.StoreState,
  PlacePayload,
  Drift.CreateWindowPayload
> = ({
  store,
  action: {
    payload: { key, name, window, location },
  },
}) => {
  if (location != "window" || selectWindowKey(store.getState()) !== MAIN_WINDOW) return;
  store.dispatch(createWindowAction(key, name, window));
};

export const closeWindowOnRemoveEffect: MiddlewareEffect<
  StoreState & Drift.StoreState,
  RemovePayload,
  Drift.CloseWindowPayload
> = ({
  store,
  action: {
    payload: { keys },
  },
}) => {
  if (selectWindowKey(store.getState()) !== MAIN_WINDOW) return;
  keys.forEach((key) => {
    const l = select(store.getState(), key);
    if (l == null || l.location === "window")
      store.dispatch(Drift.closeWindow({ key }));
  });
};

export const createWindowsOnSetProjectEffect: MiddlewareEffect<
  StoreState & Drift.StoreState,
  SetProjectPayload,
  Drift.CreateWindowPayload | Drift.CloseWindowPayload
> = ({ store }) => {
  const state = store.getState();
  const winKey = selectWindowKey(state);
  if (winKey !== MAIN_WINDOW) return;
  const { layouts } = selectSliceState(state);
  Object.values(layouts)
    .filter(({ location: l }) => l === "window")
    .forEach(({ key, name, windowProps }) => {
      if (key === Drift.MAIN_WINDOW) return;
      store.dispatch(createWindowAction(key, name, windowProps));
    });
};

const deleteLayoutsOnMosaicCloseEffect: MiddlewareEffect<
  Drift.StoreState & StoreState,
  Drift.CloseWindowPayload,
  RemovePayload
> = ({ store, action: { payload } }) => {
  if (selectWindowKey(store.getState()) !== MAIN_WINDOW) return;
  if (payload.key == null || !payload.key.startsWith(MOSAIC_WINDOW_TYPE)) return;
  const { layouts } = selectSliceState(store.getState());
  // remove all layouts associated with the mosaic window
  const layoutKeys = Object.values(layouts)
    .filter((layout) => layout.windowKey === payload.key)
    .map((layout) => layout.key);
  store.dispatch(remove({ keys: layoutKeys }));
};

export const MIDDLEWARE = [
  effectMiddleware(
    [moveMosaicTab.type, remove.type, clearProject.type, setProject.type],
    [closeWindowOnEmptyMosaicEffect],
  ),
  effectMiddleware([place.type], [createWindowOnPlaceEffect]),
  effectMiddleware([remove.type], [closeWindowOnRemoveEffect], true),
  effectMiddleware([setProject.type], [createWindowsOnSetProjectEffect]),
  effectMiddleware([Drift.closeWindow.type], [deleteLayoutsOnMosaicCloseEffect]),
];
