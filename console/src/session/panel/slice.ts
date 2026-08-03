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
  /**
   * The panels the window keeps mounted, most recently selected first. The selected
   * panel is always the head. Transient: excluded from persistence.
   */
  mounted: panel.keyZ.array().default([]),
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

// A hidden panel stops rendering but keeps streaming its channels, so the set is
// bounded. Five covers alternating between a few panels; an evicted one pays a
// remount, which costs time but no state.
const MAX_MOUNTED = 5;

// The selected panel is always the head, so eviction never takes it.
const mount = (win: WindowState, key: panel.Key): void => {
  win.mounted = [key, ...win.mounted.filter((k) => k !== key)].slice(0, MAX_MOUNTED);
};

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    select: withWindowKey<PanelKeyPayload, SliceState>((win, { payload: { key } }) => {
      win.selected = key;
      mount(win, key);
    }),
    clearSelected: withWindowKey<Window.OptionalKeyParams, SliceState>((win) => {
      win.selected = undefined;
    }),
    selectTab: withSelectedState<SelectTabPayload>(
      (pan, { payload: { tabKey, otherTabKeys } }) => {
        const next = [
          tabKey,
          ...pan.selectedTabs.filter((k) => !otherTabKeys.includes(k)),
        ];
        if (!compare.arraysEqual(pan.selectedTabs, next)) pan.selectedTabs = next;
      },
    ),
    startOverlaying: withWindowKey<Window.OptionalKeyParams, SliceState>((win) => {
      win.isOverlaid = true;
    }),
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
        win.mounted = win.mounted.filter((key) => !removed.includes(key));
        if (win.selected == null || !removed.includes(win.selected)) return;
        // Prefers the most recently used survivor, falling back to any panel the
        // window has state for: mounted is empty until the window selects again.
        const next = win.mounted[0] ?? Object.keys(win.panels).at(-1);
        win.selected = next;
        if (next != null) mount(win, next);
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

/**
 * Keeping a panel mounted is worth it once the user has switched to it, never on a
 * cold start. Clearing on write also means a project swap hydrates an empty set,
 * so a project's panels never stay mounted into the next one.
 */
export const purgeSliceState = <S extends StoreState>(state: S): S => {
  Object.values(state[SLICE_NAME].windows).forEach((win) => {
    win.mounted = [];
  });
  return state;
};

export const PERSIST_EXCLUDE = [purgeSliceState];

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

/**
 * @returns a callback that focuses the given tab and overlays it. Overlaying shows
 * the panel's focused tab, so the tab is selected first, through the same path any
 * other selection takes.
 */
export const useStartOverlaying = (panelKey?: panel.Key) => {
  const scopedPanelKey = Panel.useOptionalKey(panelKey);
  const selectTab = useSelectTab(panelKey);
  const dispatch = useDispatch<Dispatch<Action>>();
  return useCallback(
    (tabKey: panel.TabKey) => {
      if (scopedPanelKey == null) return;
      selectTab(tabKey);
      dispatch(startOverlaying({}));
    },
    [scopedPanelKey, selectTab, dispatch],
  );
};
