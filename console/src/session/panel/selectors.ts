// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Panel } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useStore } from "react-redux";

import {
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type WindowState,
  ZERO_STATE,
  ZERO_WINDOW_STATE,
} from "@/session/panel/slice";
import { Select } from "@/session/select";

/** @returns the panel slice state. */
export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export interface ActiveWindow {
  key: string;
  state: WindowState;
}

/**
 * @returns the active Drift window's key and panel state, or null when no window is
 * active. Callers that only need the state, and can render a zero value, should use
 * {@link selectWindowState}.
 */
export const selectActiveWindow = (state: StoreState): ActiveWindow | null => {
  const key = Drift.selectWindowKey(state);
  if (key == null) return null;
  return { key, state: selectSliceState(state).windows[key] ?? ZERO_WINDOW_STATE };
};

/**
 * @returns the panel state for the active Drift window, or the zero window state when
 * the window has no panel state yet.
 */
export const selectWindowState = (state: StoreState): WindowState =>
  selectActiveWindow(state)?.state ?? ZERO_WINDOW_STATE;

const selectState = (state: StoreState, key?: panel.Key): State => {
  const win = selectWindowState(state);
  key ??= win.selected;
  if (key == null) return ZERO_STATE;
  return win.panels[key] ?? ZERO_STATE;
};

/**
 * @returns a panel's selected tabs: one per leaf, most recently selected first. The
 * first key is the panel's focused tab. When key is omitted, the active window's
 * selected panel is used. Returns an empty array when the panel has no state.
 */
export const selectSelectedTabs = (
  state: StoreState,
  key?: panel.Key,
): panel.TabKey[] => selectState(state, key).selectedTabs;

/** @returns the active window's selected panel key, if any. */
export const selectSelected = (state: StoreState): panel.Key | undefined =>
  selectWindowState(state).selected;

const selectOverlaid = (state: StoreState): boolean =>
  selectWindowState(state).isOverlaid;

// The tab predicates below resolve to one bit from far more volatile inputs: a
// panel's whole selection array and the window's overlaid flag. Each is computed
// inside a single memoized selector so subscribers re-render when the answer
// flips, not whenever a sibling tab moves.

const selectIsTabFocused = (
  state: StoreState,
  key?: panel.Key,
  tabKey?: panel.TabKey,
): boolean => tabKey != null && selectSelectedTabs(state, key)[0] === tabKey;

const selectIsTabOverlaid = (
  state: StoreState,
  key?: panel.Key,
  tabKey?: panel.TabKey,
): boolean => selectOverlaid(state) && selectIsTabFocused(state, key, tabKey);

const selectIsTabVisible = (
  state: StoreState,
  key?: panel.Key,
  tabKey?: panel.TabKey,
): boolean => {
  if (tabKey == null) return false;
  const selected = selectSelectedTabs(state, key);
  if (selectOverlaid(state)) return selected[0] === tabKey;
  return selected.includes(tabKey);
};

/**
 * @returns a panel's selected tabs, as {@link selectSelectedTabs}.
 * @param key the panel to read. Defaults to the surrounding Panel scope, then to the
 * active window's selected panel.
 */
export const useSelectSelectedTabs = (key?: panel.Key): panel.TabKey[] => {
  const scoped = Panel.useOptionalKey(key);
  return Select.useMemo(
    (state: StoreState) => selectSelectedTabs(state, scoped),
    [scoped],
  );
};

/** @returns true if any tab is overlaid (focused into a modal) on the active window. */
export const useSelectOverlaid = (): boolean => Select.useMemo(selectOverlaid, []);

/** @returns a getter for whether any tab is overlaid on the active window. */
export const useGetIsOverlaid = (): (() => boolean) => {
  const store = useStore<StoreState>();
  return useCallback(() => selectOverlaid(store.getState()), [store]);
};

/**
 * @returns true if the given tab is the overlaid tab on the active window: the window
 * is overlaid and the tab is its panel's focused tab.
 * @param key the panel to read. Defaults to the surrounding Panel scope, then to the
 * window's selected panel.
 * @param tabKey the tab to check. Defaults to the surrounding Tab scope. Returns false
 * when no tab resolves.
 */
export const useSelectIsTabOverlaid = (
  key?: panel.Key,
  tabKey?: panel.TabKey,
): boolean => {
  const scoped = Panel.useOptionalKey(key);
  const scopedTab = Panel.useOptionalTabKey(tabKey);
  return Select.useMemo(
    (state: StoreState) => selectIsTabOverlaid(state, scoped, scopedTab),
    [scoped, scopedTab],
  );
};

/** @returns a getter for the active window's selected panel key, if any. */
export const useGetSelected = (): (() => panel.Key | undefined) => {
  const store = useStore<StoreState>();
  return useCallback(() => selectSelected(store.getState()), [store]);
};

/** @returns the active window's selected panel key, if any. */
export const useSelectSelected = (): panel.Key | undefined =>
  Select.useMemo(selectSelected, []);

/** @returns a getter reporting whether the active window has a selected panel. */
export const useGetIsAnySelected = (): (() => boolean) => {
  const getSelected = useGetSelected();
  return useCallback(() => getSelected() != null, [getSelected]);
};

/**
 * @returns the focused tab of a panel: the first key of its exact selection, or
 * undefined when the panel has no tabs.
 * @param key the panel to read. Defaults to the surrounding Panel scope, then to the
 * window's selected panel.
 */
export const useSelectFocusedTab = (key?: panel.Key): panel.TabKey | undefined =>
  useSelectSelectedTabs(key)[0];

/**
 * @returns a getter for the focused tab of a panel. The getter's key defaults to the
 * surrounding Panel scope, then to the window's selected panel, and returns undefined
 * when the panel has no tabs.
 */
export const useGetFocusedTab = (): ((key?: panel.Key) => panel.TabKey | undefined) => {
  const scoped = Panel.useOptionalKey();
  const store = useStore<StoreState>();
  return useCallback(
    (key: panel.Key | undefined = scoped) =>
      selectSelectedTabs(store.getState(), key)[0],
    [scoped, store],
  );
};

/**
 * @returns true if the given tab is the focused tab of its panel.
 * @param key the panel to read. Defaults to the surrounding Panel scope, then to the
 * window's selected panel.
 * @param tabKey the tab to check. Defaults to the surrounding Tab scope. Returns false
 * when no tab resolves.
 */
export const useSelectIsTabFocused = (
  key?: panel.Key,
  tabKey?: panel.TabKey,
): boolean => {
  const scoped = Panel.useOptionalKey(key);
  const scopedTab = Panel.useOptionalTabKey(tabKey);
  return Select.useMemo(
    (state: StoreState) => selectIsTabFocused(state, scoped, scopedTab),
    [scoped, scopedTab],
  );
};

/**
 * @returns a getter for whether a tab is the focused tab of its panel. Both keys
 * default to the surrounding Panel and Tab scopes. Returns false when no tab resolves.
 */
export const useGetTabIsFocused = (): ((
  key?: panel.Key,
  tabKey?: panel.TabKey,
) => boolean) => {
  const getFocusedTab = useGetFocusedTab();
  const resolvedPanel = Panel.useOptionalKey();
  const resolvedTab = Panel.useOptionalTabKey();
  return useCallback(
    (
      key: panel.Key | undefined = resolvedPanel,
      tabKey: panel.TabKey | undefined = resolvedTab,
    ) => tabKey != null && getFocusedTab(key) === tabKey,
    [getFocusedTab, resolvedPanel, resolvedTab],
  );
};

/**
 * @returns true if the given tab is rendered: it is one of its panel's selected tabs,
 * or the focused one when the window is overlaid.
 * @param key the panel to read. Defaults to the surrounding Panel scope, then to the
 * window's selected panel.
 * @param tabKey the tab to check. Defaults to the surrounding Tab scope. Returns false
 * when no tab resolves.
 */
export const useSelectIsTabVisible = (
  key?: panel.Key,
  tabKey?: panel.TabKey,
): boolean => {
  const scoped = Panel.useOptionalKey(key);
  const scopedTab = Panel.useOptionalTabKey(tabKey);
  return Select.useMemo(
    (state: StoreState) => selectIsTabVisible(state, scoped, scopedTab),
    [scoped, scopedTab],
  );
};
