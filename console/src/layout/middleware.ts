// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ActionCreatorWithPayload, type Middleware } from "@reduxjs/toolkit";
import { Drift, MAIN_WINDOW, selectWindowKey } from "@synnaxlabs/drift";
import { runtime } from "@synnaxlabs/x";

import { select, selectSliceState } from "@/layout/selectors";
import {
  clearOverlay,
  overlay,
  place,
  type PlacePayload,
  remove,
  type RemovePayload,
  setNavDrawer,
  setNavDrawerVisible,
  setProject,
  type SetProjectPayload,
  type StoreState,
  type WindowProps,
} from "@/layout/slice";
import { effectMiddleware, type MiddlewareEffect } from "@/middleware";

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

const createInjectWindowKey =
  <P extends { windowKey?: string }, T extends string = string>(
    actionCreator: ActionCreatorWithPayload<P, T>,
  ): Middleware<{}, StoreState & Drift.StoreState> =>
  (store) =>
  (next) =>
  (action) => {
    if (!actionCreator.match(action) || action.payload.windowKey != null)
      return next(action);
    const windowKey = selectWindowKey(store.getState());
    if (windowKey == null) return next(action);
    return next(actionCreator({ ...action.payload, windowKey }));
  };

export const MIDDLEWARE = [
  createInjectWindowKey(setNavDrawer),
  createInjectWindowKey(overlay),
  createInjectWindowKey(clearOverlay),
  effectMiddleware([place.type], [createWindowOnPlaceEffect]),
  effectMiddleware([remove.type], [closeWindowOnRemoveEffect], true),
  effectMiddleware([setProject.type], [createWindowsOnSetProjectEffect]),
];
