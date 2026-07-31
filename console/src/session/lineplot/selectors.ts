// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot as client } from "@synnaxlabs/client";
import { LinePlot, Panel as PPanel, type Viewport } from "@synnaxlabs/pluto";
import { type lineplot } from "@synnaxlabs/pluto/ether";
import { type record } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useStore } from "react-redux";

import {
  type ControlState,
  type SelectionState,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarState,
  type ToolbarTab,
  ZERO_STATE,
} from "@/session/lineplot/slice";
import { Panel } from "@/session/panel";
import { Select } from "@/session/select";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export interface KeyedSelectorParams extends record.Keyed<client.Key> {
  state: StoreState;
}

const createSelector = <R>(selector: (params: KeyedSelectorParams) => R) =>
  LinePlot.Scope.bindHook(({ key }: Omit<KeyedSelectorParams, "state">): R =>
    Select.useMemo((state: StoreState) => selector({ state, key }), [key]),
  );

const createGetter =
  <R>(selector: (params: KeyedSelectorParams) => R) =>
  (): ((args?: { key?: client.Key }) => R) => {
    const store = useStore<StoreState>();
    const scopeKey = LinePlot.Scope.useOptional();
    return useCallback(
      (args) =>
        selector({
          state: store.getState(),
          key: LinePlot.Scope.require(args?.key ?? scopeKey),
        }),
      [store, scopeKey],
    );
  };

const selectState = ({ state, key }: KeyedSelectorParams): State =>
  selectSliceState(state).plots[key] ?? ZERO_STATE;

export const useSelect = createSelector(selectState);
export const useGet = createGetter(selectState);

const selectToolbar = (params: KeyedSelectorParams): ToolbarState =>
  selectState(params).toolbar;

export const useSelectToolbar = createSelector(selectToolbar);
export const useGetToolbar = createGetter(selectToolbar);

export const selectActiveToolbarTab = (params: KeyedSelectorParams): ToolbarTab =>
  selectToolbar(params).activeTab;

export const useSelectActiveToolbarTab = createSelector(selectActiveToolbarTab);
export const useGetActiveToolbarTab = createGetter(selectActiveToolbarTab);

export const selectControlState = (params: KeyedSelectorParams): ControlState =>
  selectState(params).control;

export const useSelectControlState = createSelector(selectControlState);

const selectViewportMode = (params: KeyedSelectorParams): Viewport.Mode =>
  selectState(params).mode;

export const useSelectViewportMode = createSelector(selectViewportMode);
export const useGetViewportMode = createGetter(selectViewportMode);

const selectHiddenLines = (params: KeyedSelectorParams): string[] =>
  selectState(params).hiddenLines;

export const useSelectHiddenLines = createSelector(selectHiddenLines);

const selectMeasureMode = (params: KeyedSelectorParams): lineplot.measure.Mode =>
  selectState(params).measure.mode;

export const useSelectMeasureMode = createSelector(selectMeasureMode);

const selectSelection = (params: KeyedSelectorParams): SelectionState =>
  selectState(params).selection;

export const useSelectSelection = createSelector(selectSelection);
export const useGetSelection = createGetter(selectSelection);

export const selectSelectedRules = (params: KeyedSelectorParams): string[] =>
  selectState(params).selectedRules;

export const useSelectSelectedRules = createSelector(selectSelectedRules);

const selectAnnotationsVisible = (params: KeyedSelectorParams): boolean =>
  selectState(params).annotations.visible;

export const useSelectAnnotationsVisible = createSelector(selectAnnotationsVisible);

export const useGetFocusedKey = (): (() => client.Key | undefined) => {
  const getSelectedPanel = Panel.useGetSelected();
  const getFocusedTabKey = Panel.useGetFocusedTab();
  const getTab = PPanel.useGetTab();
  return useCallback(() => {
    const panelKey = getSelectedPanel();
    const tabKey = getFocusedTabKey();
    if (panelKey == null || tabKey == null) return undefined;
    const tab = getTab({ key: panelKey, tabKey });
    if (tab.variant === "resource" && tab.resource.type === "lineplot")
      return tab.resource.key;
    return undefined;
  }, [getSelectedPanel, getFocusedTabKey, getTab]);
};
