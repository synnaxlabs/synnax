// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { selectSliceState } from "@/layered/session/lineplot/selectors";
import {
  remove,
  type RemovePayload,
  type StoreState,
} from "@/layered/session/lineplot/slice";
import { Layout } from "@/layout";
import { effectMiddleware, type MiddlewareEffect } from "@/middleware";

// deleteEffect drops session state for plots whose layout has been removed (tab closed)
// or whose project no longer includes them. The line plot's document lives on the server;
// this only garbage-collects the per-plot Console UI state so it does not accumulate.
export const deleteEffect: MiddlewareEffect<
  Layout.StoreState & StoreState,
  Layout.RemovePayload | Layout.SetProjectPayload,
  RemovePayload
> = ({ action, store }) => {
  const state = store.getState();
  const lineState = selectSliceState(state);
  const layout = Layout.selectSliceState(state);
  const keys = "keys" in action.payload ? action.payload.keys : [];
  const toRemove = Object.keys(lineState.plots).filter(
    (p) => layout.layouts[p] == null || keys.includes(p),
  );
  if (toRemove.length > 0) store.dispatch(remove({ keys: toRemove }));
};

export const MIDDLEWARE = [
  effectMiddleware([Layout.remove.type, Layout.setProject.type], [deleteEffect]),
];
