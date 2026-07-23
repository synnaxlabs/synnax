// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch } from "@reduxjs/toolkit";
import { panel } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Synnax } from "@synnaxlabs/pluto";
import { useCallback, useEffect } from "react";
import { useDispatch, useSelector, useStore } from "react-redux";

import { purge, reconcileSelection } from "@/session/panel/slice";
// Type-only: a value import of the store would cycle through its domain barrels.
import type { Action, State } from "@/session/store";
import { Synchronizer } from "@/session/synchronizer";

const selectKeys = (state: State): string[] => {
  const keys = new Set<string>();
  Object.values(state.panels.windows).forEach((win) => {
    if (win.selected != null) keys.add(win.selected);
    Object.keys(win.panels).forEach((key) => keys.add(key));
  });
  return [...keys];
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers = {
  usePruneDeletedPanels: Synchronizer.create({
    onDelete: (client, handler) => client.panels.onDelete(handler),
    retrieveExisting: async (client, keys) =>
      (await client.panels.retrieve({ keys })).map(({ key }) => key),
    selectKeys,
    remove: purge,
  }),
};

const selectActiveWindowSelected = (state: State): panel.Key | undefined => {
  const windowKey = Drift.selectWindowKey(state);
  if (windowKey == null) return undefined;
  return state.panels.windows[windowKey]?.selected;
};

// Converges the active window's tab selection to each referenced panel's live
// tree. Runs per window off its own cache feed, so the selection and the tree
// update in the same tick; cross-window echoes of the action are no-ops.
const useReconcileTabSelections = (): void => {
  const dispatch = useDispatch<Dispatch<Action>>();
  const store = useStore<State>();
  const client = Synnax.use();
  const reconcile = useCallback(
    (pan: panel.Panel) => {
      const state = store.getState();
      const windowKey = Drift.selectWindowKey(state);
      if (windowKey == null) return;
      const win = state.panels.windows[windowKey];
      if (win == null) return;
      if (win.selected !== pan.key && win.panels[pan.key] == null) return;
      dispatch(
        reconcileSelection({
          windowKey,
          key: pan.key,
          leaves: panel.leafTabGroups(pan.root),
        }),
      );
    },
    [dispatch, store],
  );
  useEffect(() => {
    if (client == null) return;
    return client.panels.onSet(reconcile);
  }, [client, reconcile]);
  // A panel cached before the window first selects it fires no set event, so the
  // selection change itself triggers a reconcile against the cached tree.
  const selected = useSelector(selectActiveWindowSelected);
  useEffect(() => {
    if (client == null || selected == null) return;
    const cached = client.panels.getCached(selected);
    if (cached == null || cached.variant === "deleted") return;
    reconcile(cached.data);
  }, [client, selected, reconcile]);
};

export const WINDOW_SYNCHRONIZERS: Synchronizer.Synchronizers = {
  useReconcileTabSelections,
};
