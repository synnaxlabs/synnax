// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type Dispatch, type PayloadAction } from "@reduxjs/toolkit";
import { NotFoundError, panel, query } from "@synnaxlabs/client";
import { type Drift } from "@synnaxlabs/drift";
import { Panel, Synnax } from "@synnaxlabs/pluto";
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
   * The panels the window keeps mounted, most recently selected first. Transient:
   * excluded from persistence, so a hydrated window starts empty with a panel still
   * selected. selectMounted folds the selection back in.
   */
  mounted: panel.keyZ.array().default([]),
});

export interface WindowState extends z.output<typeof windowStateZ> {}

export const ZERO_WINDOW_STATE: WindowState = windowStateZ.parse({});

export const sliceStateZ = z.object({
  windows: z.record(z.string(), windowStateZ).default({}),
  /**
   * The strip's panel order, shared by every window. Reconciliation appends new
   * panels name-sorted and prunes deleted ones; only an explicit reorder moves an
   * entry, so a panel never changes slot behind the user's back.
   */
  order: panel.keyZ.array().default([]),
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

export interface OrderEntry {
  key: panel.Key;
  name: string;
}

export interface ReconcileOrderPayload {
  panels: OrderEntry[];
}

export interface ReorderPayload {
  key: panel.Key;
  index: number;
}

const withWindowKey = Window.createWithKeyHandler(windowStateZ);

const panelState = (win: WindowState, key: panel.Key): State =>
  (win.panels[key] ??= stateZ.parse({}));

const withSelectedState = <Payload extends PanelKeyPayload>(
  handler: (
    state: State,
    action: PayloadAction<require.Require<Payload, "windowKey">>,
  ) => void,
) =>
  withWindowKey<Payload, SliceState>((win, action) => {
    handler(panelState(win, action.payload.key), action);
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
        // Skip the no-op reselect fired on every click inside a focused tab's content;
        // a state change here re-renders mid-click and eats checkbox toggles.
        if (!compare.arraysEqual(pan.selectedTabs, next)) pan.selectedTabs = next;
      },
    ),
    startOverlaying: withWindowKey<Window.OptionalKeyParams, SliceState>((win) => {
      win.isOverlaid = true;
    }),
    // reconcileSelection converges a panel's selection to its live tree: one tab
    // per leaf, most recent first; a leaf with no selected tab contributes its
    // last tab.
    reconcileSelection: withWindowKey<ReconcileSelectionPayload, SliceState>(
      (win, { payload: { key, leaves } }) => {
        const pan = panelState(win, key);
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
        // Overlaying shows the selected panel's focused tab. Losing that tab ends the
        // overlay; the flag would otherwise pull in whichever tab is focused next, or
        // the next tab created in an emptied panel.
        const focused = pan.selectedTabs[0];
        if (win.selected === key && focused != null && !leafOf.has(focused))
          win.isOverlaid = false;
        if (!compare.arraysEqual(pan.selectedTabs, next)) pan.selectedTabs = next;
      },
    ),
    stopOverlaying: withWindowKey<Window.OptionalKeyParams, SliceState>((win) => {
      win.isOverlaid = false;
    }),
    // reconcileOrder converges the order to the project's live membership: deleted
    // panels prune, unknown panels append name-sorted. On first sight of a project
    // the whole order materializes name-sorted through the same append path.
    reconcileOrder: (
      state,
      { payload: { panels } }: PayloadAction<ReconcileOrderPayload>,
    ) => {
      const live = new Set(panels.map(({ key }) => key));
      const kept = state.order.filter((key) => live.has(key));
      const fresh = panels
        .filter(({ key }) => !kept.includes(key))
        .sort((a, b) => compare.stringsWithNumbers(a.name, b.name))
        .map(({ key }) => key);
      const next = [...kept, ...fresh];
      if (!compare.arraysEqual(state.order, next)) state.order = next;
    },
    // The index is a strip insertion slot resolved with the panel still in place,
    // so a move toward the end lands one slot short of the raw index.
    reorder: (state, { payload: { key, index } }: PayloadAction<ReorderPayload>) => {
      const from = state.order.indexOf(key);
      const next = [...state.order];
      if (from !== -1) next.splice(from, 1);
      const to = Math.min(from !== -1 && from < index ? index - 1 : index, next.length);
      next.splice(Math.max(to, 0), 0, key);
      if (!compare.arraysEqual(state.order, next)) state.order = next;
    },
    remove: (state, { payload: keys }: PayloadAction<RemovePayload>) => {
      const removed = array.toArray(keys);
      state.order = state.order.filter((key) => !removed.includes(key));
      Object.values(state.windows).forEach((win) => {
        removed.forEach((key) => delete win.panels[key]);
        win.mounted = win.mounted.filter((key) => !removed.includes(key));
        if (win.selected == null || !removed.includes(win.selected)) return;
        // The overlaid tab belongs to the selected panel, so losing it ends the
        // overlay.
        win.isOverlaid = false;
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
  reconcileOrder,
  reconcileSelection,
  reorder,
  reset,
} = actions;

export {
  clearSelected,
  internalSelectTab,
  reconcileOrder,
  reconcileSelection,
  reducer,
  remove,
  reorder,
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
  const client = Synnax.use();
  const dispatch = useDispatch<Dispatch<Action>>();
  return useCallback(
    (key: panel.TabKey, overridePanelKey?: panel.Key) => {
      const resolvedPanelKey = overridePanelKey ?? scopedPanelKey;
      if (resolvedPanelKey == null) return;
      const cached = client?.panels.getCached(resolvedPanelKey);
      if (!query.isLive(cached))
        throw new NotFoundError(`Panel with key ${resolvedPanelKey} not found`);
      const leaf = panel.findTabLeaf(cached.root, key);
      if (leaf == null)
        throw new NotFoundError(
          `Leaf holding tab ${key} not found in panel ${resolvedPanelKey}`,
        );
      dispatch(
        internalSelectTab({
          tabKey: key,
          key: resolvedPanelKey,
          otherTabKeys: leaf.tabs.map((t) => t.key),
        }),
      );
    },
    [scopedPanelKey, client, dispatch],
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
