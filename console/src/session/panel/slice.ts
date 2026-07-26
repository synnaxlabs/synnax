// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type Dispatch, type PayloadAction } from "@reduxjs/toolkit";
import { panel } from "@synnaxlabs/client";
import { type Drift } from "@synnaxlabs/drift";
import { Panel } from "@synnaxlabs/pluto";
import { array, compare, type require } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";
import z from "zod";

import { Window } from "@/session/window";

export const stateZ = z.object({
  selectedTabs: panel.tabKeyZ.array().default([]),
});

export interface State extends z.output<typeof stateZ> {}

export const ZERO_STATE = stateZ.parse({});

export const windowStateZ = z.object({
  selected: panel.keyZ.optional(),
  isOverlaid: z.boolean().optional().default(false),
  panels: z.record(panel.keyZ, stateZ).default({}),
});

export interface WindowState extends z.output<typeof windowStateZ> {}

export const ZERO_WINDOW_STATE: WindowState = windowStateZ.parse({});

export const sliceStateZ = z.object({
  windows: z.record(z.string(), windowStateZ).default({}),
});

export interface SliceState extends z.output<typeof sliceStateZ> {}

export interface StoreState extends Drift.StoreState {
  [SLICE_NAME]: SliceState;
}

export const SLICE_NAME = "panels";

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({});

export interface PanelKeyPayload extends Window.OptionalKeyParams {
  key: panel.Key;
}

export interface TabAndPanelKeyPayload extends PanelKeyPayload {
  tabKey: string;
}

export type RemovePayload = panel.Key | panel.Key[];

interface SelectTabPayload extends TabAndPanelKeyPayload {
  otherTabKeys: panel.TabKey[];
}

// Overlaying shows the panel's focused tab, so naming a tab moves it to the head of
// the selection. The panel key defaults to the window's selected panel.
export interface StartOverlayingPayload extends Window.OptionalKeyParams {
  key?: panel.Key;
  tabKey?: panel.TabKey;
  otherTabKeys?: panel.TabKey[];
}

export interface ReconcileSelectionPayload extends PanelKeyPayload {
  leaves: panel.TabKey[][];
}

const withWindowKey = Window.createWithKeyHandler(windowStateZ);

const withSelectedState = <Payload extends PanelKeyPayload>(
  handler: (
    state: State,
    action: PayloadAction<require.Require<Payload, "windowKey">>,
  ) => void,
) =>
  withWindowKey<Payload, SliceState>((win, action) => {
    const { key } = action.payload;
    let pan = win.panels[key];
    if (pan == null) {
      pan = stateZ.parse({});
      win.panels[key] = pan;
    }
    handler(pan, action);
  });

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    select: withWindowKey<PanelKeyPayload, SliceState>((win, { payload: { key } }) => {
      win.selected = key;
    }),
    clearSelected: withWindowKey<Window.OptionalKeyParams, SliceState>((win) => {
      win.selected = undefined;
    }),
    selectTab: withSelectedState<SelectTabPayload>(
      (pan, { payload: { tabKey, otherTabKeys } }) => {
        pan.selectedTabs = [
          tabKey,
          ...pan.selectedTabs.filter((k) => !otherTabKeys.includes(k)),
        ];
      },
    ),
    startOverlaying: withWindowKey<StartOverlayingPayload, SliceState>(
      (win, { payload: { key = win.selected, tabKey, otherTabKeys = [] } }) => {
        win.isOverlaid = true;
        if (tabKey == null || key == null) return;
        const pan = (win.panels[key] ??= stateZ.parse({}));
        pan.selectedTabs = [
          tabKey,
          ...pan.selectedTabs.filter((k) => k !== tabKey && !otherTabKeys.includes(k)),
        ];
      },
    ),
    // reconcileSelection converges a panel's selection to its live tree: one tab
    // per leaf, most recent first; a leaf with no selected tab contributes its
    // last tab.
    reconcileSelection: withSelectedState<ReconcileSelectionPayload>(
      (pan, { payload: { leaves } }) => {
        const leafOf = new Map<panel.TabKey, number>();
        leaves.forEach((tabs, i) => tabs.forEach((tab) => leafOf.set(tab, i)));
        const claimed = new Set<number>();
        const next: panel.TabKey[] = [];
        pan.selectedTabs.forEach((k) => {
          const leaf = leafOf.get(k);
          if (leaf == null || claimed.has(leaf)) return;
          claimed.add(leaf);
          next.push(k);
        });
        leaves.forEach((tabs, i) => {
          if (!claimed.has(i) && tabs.length > 0) next.push(tabs[tabs.length - 1]);
        });
        if (!compare.arraysEqual(pan.selectedTabs, next)) pan.selectedTabs = next;
      },
    ),
    stopOverlaying: withWindowKey<Window.OptionalKeyParams, SliceState>((win) => {
      win.isOverlaid = false;
    }),
    remove: (state, { payload: keys }: PayloadAction<RemovePayload>) => {
      const removed = array.toArray(keys);
      Object.values(state.windows).forEach((win) => {
        removed.forEach((key) => delete win.panels[key]);
        if (win.selected != null && removed.includes(win.selected))
          win.selected = undefined;
      });
    },
    reset: () => ZERO_SLICE_STATE,
  },
});

const {
  select,
  clearSelected,
  remove,
  selectTab: internalSelectTab,
  startOverlaying,
  stopOverlaying,
  reconcileSelection,
  reset,
} = actions;

export {
  clearSelected,
  internalSelectTab,
  reconcileSelection,
  reducer,
  remove,
  reset,
  select,
  startOverlaying,
  stopOverlaying,
};

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];

export const MIDDLEWARE = [
  Window.createInjectKeyMiddleware([
    select,
    clearSelected,
    internalSelectTab,
    startOverlaying,
    stopOverlaying,
    reconcileSelection,
  ]),
];

export const PERSIST_EXCLUDE = [];

export const useSelectTab = (panelKey?: panel.Key) => {
  const scopedPanelKey = Panel.useOptionalKey(panelKey);
  const getTabLeaf = Panel.useGetTabLeaf();
  const dispatch = useDispatch<Dispatch<Action>>();
  return useCallback(
    (key: panel.TabKey, overridePanelKey?: panel.Key) => {
      const resolvedPanelKey = overridePanelKey ?? scopedPanelKey;
      if (resolvedPanelKey == null) return;
      const leaf = getTabLeaf({ key: resolvedPanelKey, tabKey: key });
      dispatch(
        internalSelectTab({
          tabKey: key,
          key: resolvedPanelKey,
          otherTabKeys: leaf.tabs.map((t) => t.key),
        }),
      );
    },
    [scopedPanelKey, getTabLeaf, dispatch],
  );
};

/** @returns a callback that overlays the given tab, evicting its leaf siblings from
 * the selection so exactly one tab stays selected per leaf. */
export const useStartOverlaying = (panelKey?: panel.Key) => {
  const scopedPanelKey = Panel.useOptionalKey(panelKey);
  const getTabLeaf = Panel.useGetTabLeaf();
  const dispatch = useDispatch<Dispatch<Action>>();
  return useCallback(
    (tabKey: panel.TabKey) => {
      if (scopedPanelKey == null) return;
      const leaf = getTabLeaf({ key: scopedPanelKey, tabKey });
      dispatch(
        startOverlaying({
          key: scopedPanelKey,
          tabKey,
          otherTabKeys: leaf.tabs.map((t) => t.key),
        }),
      );
    },
    [scopedPanelKey, getTabLeaf, dispatch],
  );
};
