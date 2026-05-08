// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Middleware, type PayloadAction } from "@reduxjs/toolkit";
import { type Drift } from "@synnaxlabs/drift";

import { selectSliceState } from "@/arc/selectors";
import {
  remove,
  type RemovePayload,
  setSelected,
  type SetSelectedPayload,
  type StoreState,
} from "@/arc/slice";
import { Layout } from "@/layout";
import { effectMiddleware, type MiddlewareEffect } from "@/middleware";

export const deleteEffect: MiddlewareEffect<
  Layout.StoreState & StoreState,
  Layout.RemovePayload | Layout.SetWorkspacePayload,
  RemovePayload
> = ({ action, store }) => {
  const state = store.getState();
  const slateSlice = selectSliceState(state);
  const layoutSlice = Layout.selectSliceState(state);
  // This is the case where the action does an explicit removal.
  const keys = "keys" in action.payload ? action.payload.keys : [];
  // We also just do a general purpose garbage collection if necessary.
  const toRemove = Object.keys(slateSlice.arcs).filter(
    (p) => keys.includes(p) || layoutSlice.layouts[p] == null,
  );
  if (toRemove.length > 0) store.dispatch(remove({ keys: toRemove }));
};

const filterSelectionMiddleware: Middleware<
  object,
  Layout.StoreState & Drift.StoreState & StoreState
> = (store) => (next) => (action) => {
  const a = action as PayloadAction<SetSelectedPayload>;
  if (a.type !== setSelected.type) return next(action);
  const state = store.getState();
  const layout = Layout.select(state, a.payload.key);
  if (layout == null) return next(action);
  const activeKey = Layout.selectActiveMosaicTabKeyAndNotBlurred(
    state,
    layout.windowKey,
  );
  if (activeKey !== a.payload.key) return;
  return next(action);
};

export const MIDDLEWARE = [
  effectMiddleware([Layout.remove.type, Layout.setWorkspace.type], [deleteEffect]),
  filterSelectionMiddleware,
];
