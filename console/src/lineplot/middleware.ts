// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { selectSliceState } from "@/lineplot/selectors";
import { remove, type RemovePayload, type StoreState } from "@/lineplot/slice";
import { effectMiddleware, type MiddlewareEffect } from "@/middleware";

export const deleteEffect: MiddlewareEffect<
  Layout.StoreState & StoreState,
  Layout.RemovePayload | Layout.SetWorkspacePayload,
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
  effectMiddleware([Layout.remove.type, Layout.setWorkspace.type], [deleteEffect]),
];
