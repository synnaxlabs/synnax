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

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const useSelectSliceState = (): SliceState =>
  Select.useMemo((state: StoreState) => selectSliceState(state), []);

export const selectWindowState = (state: RequiredStoreState): WindowState => {
  const windowKey = Drift.selectWindowKey(state);
  if (windowKey == null) return ZERO_WINDOW_STATE;
  return selectSliceState(state).windows[windowKey] ?? ZERO_WINDOW_STATE;
};

export const selectSelectedTabs = (
  state: RequiredStoreState,
  panelKey: panel.Key,
): panel.TabKey[] => selectState(state, panelKey).selectedTabs;

export const useSelectSelectedTabs = (key?: panel.Key): panel.TabKey[] => {
  const resolved = Panel.useKey(key);
  const tabKeys = Select.useMemo(
    (state: RequiredStoreState) => selectSelectedTabs(state, resolved),
    [resolved],
  );
  return Panel.useSelectFilteredTabKeys({ tabKeys, key: resolved });
};

const selectState = (state: RequiredStoreState, panelKey: panel.Key): State =>
  selectWindowState(state).panels[panelKey] ?? ZERO_STATE;

const selectOverlaid = (state: RequiredStoreState): boolean =>
  selectWindowState(state).isOverlaid;

export const useSelectOverlaid = (): boolean =>
  Select.useMemo((state: RequiredStoreState) => selectOverlaid(state), []);

export const useGetIsOverlaid = (): (() => boolean) => {
  const store = useStore<RequiredStoreState>();
  return useCallback(() => selectOverlaid(store.getState()), [store]);
};

export const useSelectIsTabOverlaid = (
  panelKey?: panel.Key,
  tabKey?: panel.TabKey,
): boolean => {
  const resolvedPanelKey = Panel.useKey(panelKey);
  const resolvedTabKey = Panel.useTabKey(tabKey);
  return Select.useMemo(
    (state: RequiredStoreState) =>
      selectOverlaid(state) &&
      selectSelectedTabs(state, resolvedPanelKey)[0] === resolvedTabKey,
    [resolvedPanelKey, resolvedTabKey],
  );
};

export const useSelectIsTabFocused = (
  panelKey?: panel.Key,
  tabKey?: panel.TabKey,
): boolean => {
  const resolvedPanelKey = Panel.useKey(panelKey);
  const resolvedTabKey = Panel.useTabKey(tabKey);
  return Select.useMemo(
    (state: RequiredStoreState) =>
      selectSelectedTabs(state, resolvedPanelKey)[0] === resolvedTabKey,
    [resolvedPanelKey, resolvedTabKey],
  );
};

export const selectSelected = (state: RequiredStoreState): panel.Key | undefined =>
  selectWindowState(state).selected;

export const useGetSelected = (): (() => panel.Key | undefined) => {
  const store = useStore<RequiredStoreState>();
  return useCallback(() => selectSelected(store.getState()), [store]);
};

export const useSelectSelected = (): panel.Key | undefined =>
  Select.useMemo(selectSelected, []);

export const useSelectIsAnyTabFocused = (key?: panel.Key): boolean =>
  useSelectSelectedTabs(key).length > 0;

export const useSelectFocusedTab = (key?: panel.Key): panel.TabKey | undefined =>
  useSelectSelectedTabs(key)[0];

export const useGetSelectedTabs = (key?: panel.Key): (() => panel.TabKey[]) => {
  const resolved = Panel.useOptionalKey(key);
  const store = useStore<RequiredStoreState>();
  const filter = Panel.useGetFilteredTabKeys();
  return useCallback(() => {
    if (resolved == null) return [];
    return filter({
      key: resolved,
      tabKeys: selectSelectedTabs(store.getState(), resolved),
    });
  }, [resolved, store, filter]);
};

export const useGetFocusedTab = (
  panelKey?: panel.Key,
): (() => panel.TabKey | undefined) => {
  const getSelected = useGetSelectedTabs(panelKey);
  return useCallback(() => getSelected()[0], [getSelected]);
};

export const useGetTabIsFocused = (
  pabelKey?: panel.Key,
  tabKey?: panel.TabKey,
): (() => boolean) => {
  const getFocused = useGetFocusedTab(pabelKey);
  const resolvedTabKey = Panel.useTabKey(tabKey);
  return useCallback(
    () => getFocused() === resolvedTabKey,
    [getFocused, resolvedTabKey],
  );
};

const selectIsAnySelected = (state: RequiredStoreState): boolean =>
  selectWindowState(state).selected != null;

export const useGetIsAnySelected = (): (() => boolean) => {
  const store = useStore<RequiredStoreState>();
  return useCallback(() => selectIsAnySelected(store.getState()), [store]);
};
