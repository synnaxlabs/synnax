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

interface RequiredStoreState extends StoreState, Drift.StoreState {}

/** @returns the panel slice state. */
export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

/**
 * @returns the panel state for the active Drift window, or the zero window state when
 * the window has no panel state yet.
 */
export const selectWindowState = (state: RequiredStoreState): WindowState => {
  const windowKey = Drift.selectWindowKey(state);
  if (windowKey == null) return ZERO_WINDOW_STATE;
  return selectSliceState(state).windows[windowKey] ?? ZERO_WINDOW_STATE;
};

const selectState = (state: RequiredStoreState, key?: panel.Key): State => {
  const win = selectWindowState(state);
  key ??= win.selected;
  if (key == null) return ZERO_STATE;
  return win.panels[key] ?? ZERO_STATE;
};

/**
 * @returns the ordered selected tabs (most recently focused first) for a panel. When
 * key is omitted, the active window's selected panel is used. Returns an empty array
 * when the panel has no state. The result is not filtered against the panel's live
 * tree, so it may contain tabs that have since been removed.
 */
export const selectSelectedTabs = (
  state: RequiredStoreState,
  key?: panel.Key,
): panel.TabKey[] => selectState(state, key).selectedTabs;

/** @returns the active window's selected panel key, if any. */
export const selectSelected = (state: RequiredStoreState): panel.Key | undefined =>
  selectWindowState(state).selected;

const selectOverlaid = (state: RequiredStoreState): boolean =>
  selectWindowState(state).isOverlaid;

/**
 * @returns the selected tabs for a panel that still exist in the panel's live tree,
 * with removed tabs filtered out.
 * @param key the panel to read. Defaults to the surrounding Panel scope. Must resolve
 * to a panel via the argument or a surrounding Panel scope.
 */
export const useSelectSelectedTabs = (key?: panel.Key): panel.TabKey[] => {
  const resolved = Panel.useOptionalKey(key);
  const tabKeys = Select.useMemo(
    (state: RequiredStoreState) => selectSelectedTabs(state, resolved),
    [resolved],
  );
  return Panel.useSelectFilteredTabKeys({ tabKeys, key: resolved });
};

/** @returns true if any tab is overlaid (focused into a modal) on the active window. */
export const useSelectOverlaid = (): boolean =>
  Select.useMemo((state: RequiredStoreState) => selectOverlaid(state), []);

/** @returns a getter for whether any tab is overlaid on the active window. */
export const useGetIsOverlaid = (): (() => boolean) => {
  const store = useStore<RequiredStoreState>();
  return useCallback(() => selectOverlaid(store.getState()), [store]);
};

/**
 * @returns true if the given tab is the overlaid tab on the active window: the window
 * is overlaid and the tab is its panel's focused (first selected) tab.
 * @param key the panel to read. Defaults to the surrounding Panel scope, then to the
 * window's selected panel.
 * @param tabKey the tab to check. Defaults to the surrounding Tab scope. Returns false
 * when no tab resolves.
 */
export const useSelectIsTabOverlaid = (
  key?: panel.Key,
  tabKey?: panel.TabKey,
): boolean => {
  key = Panel.useOptionalKey(key);
  tabKey = Panel.useOptionalTabKey(tabKey);
  return Select.useMemo(
    (state: RequiredStoreState) =>
      tabKey != null &&
      selectOverlaid(state) &&
      selectSelectedTabs(state, key)[0] === tabKey,
    [key, tabKey],
  );
};

/** @returns a getter for the active window's selected panel key, if any. */
export const useGetSelected = (): (() => panel.Key | undefined) => {
  const store = useStore<RequiredStoreState>();
  return useCallback(() => selectSelected(store.getState()), [store]);
};

/** @returns the active window's selected panel key, if any. */
export const useSelectSelected = (): panel.Key | undefined =>
  Select.useMemo(selectSelected, []);

/**
 * @returns the focused (first selected) tab of a panel, or undefined when the panel
 * has no selected tabs.
 * @param key the panel to read. Defaults to the surrounding Panel scope, then to the
 * window's selected panel.
 */
export const useSelectFocusedTab = (key?: panel.Key): panel.TabKey | undefined => {
  const resolved = Panel.useOptionalKey(key);
  return Select.useMemo(
    (state: RequiredStoreState) => selectSelectedTabs(state, resolved)[0],
    [resolved],
  );
};

/**
 * @returns a getter for the focused (first selected) tab of a panel. The getter's key
 * defaults to the surrounding Panel scope, then to the window's selected panel, and
 * returns undefined when the panel has no selected tabs.
 */
export const useGetFocusedTab = (): ((key?: panel.Key) => panel.TabKey | undefined) => {
  const resolved = Panel.useOptionalKey();
  const store = useStore<RequiredStoreState>();
  return useCallback(
    (key: panel.Key | undefined = resolved) =>
      selectSelectedTabs(store.getState(), key)[0],
    [resolved, store],
  );
};

/**
 * @returns true if the given tab is the focused (first selected) tab of its panel.
 * @param key the panel to read. Defaults to the surrounding Panel scope, then to the
 * window's selected panel.
 * @param tabKey the tab to check. Defaults to the surrounding Tab scope. Returns false
 * when no tab resolves.
 */
export const useSelectIsTabFocused = (
  key?: panel.Key,
  tabKey?: panel.TabKey,
): boolean => {
  key = Panel.useOptionalKey(key);
  tabKey = Panel.useOptionalTabKey(tabKey);
  return Select.useMemo(
    (state: RequiredStoreState) =>
      tabKey != null && selectSelectedTabs(state, key)[0] === tabKey,
    [key, tabKey],
  );
};

/**
 * @returns a getter for whether a tab is the focused (first selected) tab of its
 * panel. Both keys default to the surrounding Panel and Tab scopes. Returns false when
 * no tab resolves.
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
