// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift, MAIN_WINDOW, selectWindowKey } from "@synnaxlabs/drift";
import { Mosaic } from "@synnaxlabs/pluto";
import { runtime } from "@synnaxlabs/x";

import { Layout } from "@/layout";
import { select, selectSliceState } from "@/layout/selectors";
import {
  clearWorkspace,
  MOSAIC_WINDOW_TYPE,
  moveMosaicTab,
  type MoveMosaicTabPayload,
  place,
  type PlacePayload,
  remove,
  type RemovePayload,
  setWorkspace,
  type SetWorkspacePayload,
  type StoreState,
  type WindowProps,
} from "@/layout/slice";
import { effectMiddleware, type MiddlewareEffect } from "@/middleware";

export const closeWindowOnEmptyMosaicEffect: MiddlewareEffect<
  StoreState & Drift.StoreState,
  MoveMosaicTabPayload | RemovePayload,
  Drift.CloseWindowPayload
> = ({ getState, dispatch }) => {
  const s = getState();
  if (selectWindowKey(s) !== MAIN_WINDOW) return;
  const { mosaics } = selectSliceState(s);
  // Close windows with empty mosaics.
  Object.entries(mosaics).forEach(([k, { root }]) => {
    if (k === Drift.MAIN_WINDOW || !Mosaic.isEmpty(root)) return;
    const win = Drift.selectWindow(s, k);
    if (win != null) dispatch(Drift.closeWindow({ key: k }));
  });
  // Close windows whose mosaics no longer exist.
  const windows = Drift.selectWindows(s);
  windows.forEach((win) => {
    if (!win.key.startsWith(MOSAIC_WINDOW_TYPE) || win.key in mosaics) return;
    dispatch(Drift.closeWindow({ key: win.key }));
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
  getState,
  dispatch,
  action: {
    payload: { key, name, window, location },
  },
}) => {
  if (location != "window" || selectWindowKey(getState()) !== MAIN_WINDOW) return;
  dispatch(createWindowAction(key, name, window));
};

export const closeWindowOnRemoveEffect: MiddlewareEffect<
  StoreState & Drift.StoreState,
  RemovePayload,
  Drift.CloseWindowPayload
> = ({
  getState,
  action: {
    payload: { keys },
  },
  dispatch,
}) => {
  if (selectWindowKey(getState()) !== MAIN_WINDOW) return;
  keys.forEach((key) => {
    const l = select(getState(), key);
    if (l == null || l.location === "window") dispatch(Drift.closeWindow({ key }));
  });
};

export const createWindowsOnSetWorkspaceEffect: MiddlewareEffect<
  StoreState & Drift.StoreState,
  SetWorkspacePayload,
  Drift.CreateWindowPayload | Drift.CloseWindowPayload
> = ({ getState, dispatch }) => {
  const state = getState();
  const winKey = selectWindowKey(state);
  if (winKey !== MAIN_WINDOW) return;
  const { layouts } = selectSliceState(state);
  Object.values(layouts)
    .filter(({ location: l }) => l === "window")
    .forEach(({ key, name, window }) => {
      if (key === Drift.MAIN_WINDOW) return;
      dispatch(createWindowAction(key, name, window));
    });
};

const deleteLayoutsOnMosaicCloseEffect: MiddlewareEffect<
  Drift.StoreState & StoreState,
  Drift.CloseWindowPayload,
  RemovePayload
> = ({ getState, action: { payload }, dispatch }) => {
  if (selectWindowKey(getState()) !== MAIN_WINDOW) return;
  const s = getState();
  if (payload.key == null || !payload.key.startsWith(Layout.MOSAIC_WINDOW_TYPE)) return;
  const { layouts } = Layout.selectSliceState(s);
  // remove all layouts associated with the mosaic window
  const layoutKeys = Object.values(layouts)
    .filter((layout) => layout.windowKey === payload.key)
    .map((layout) => layout.key);
  dispatch(Layout.remove({ keys: layoutKeys }));
};

export const MIDDLEWARE = [
  effectMiddleware(
    [moveMosaicTab.type, remove.type, clearWorkspace.type, setWorkspace.type],
    [closeWindowOnEmptyMosaicEffect],
  ),
  effectMiddleware([place.type], [createWindowOnPlaceEffect]),
  effectMiddleware([remove.type], [closeWindowOnRemoveEffect], true),
  effectMiddleware([setWorkspace.type], [createWindowsOnSetWorkspaceEffect]),
  effectMiddleware([Drift.closeWindow.type], [deleteLayoutsOnMosaicCloseEffect]),
];
