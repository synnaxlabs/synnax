// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot as client, panel, query } from "@synnaxlabs/client";
import { LinePlot, Panel as PPanel, Synnax, type Viewport } from "@synnaxlabs/pluto";
import { type lineplot } from "@synnaxlabs/pluto/ether";
import { type record } from "@synnaxlabs/x";

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
import { Window } from "@/session/window";

const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export interface KeyedSelectorParams extends record.Keyed<client.Key> {
  state: StoreState;
}

const createSelector = <R>(selector: (params: KeyedSelectorParams) => R) => {
  const useSelector = ({ key }: Omit<KeyedSelectorParams, "state">): R =>
    Select.useMemo((state: StoreState) => selector({ state, key }), [key]);
  return LinePlot.Scope.bindHook(useSelector);
};

const selectState = ({ state, key }: KeyedSelectorParams): State =>
  Window.selectDocument(state, selectSliceState(state), key) ?? ZERO_STATE;

export const useSelect = createSelector(selectState);

const selectToolbar = (params: KeyedSelectorParams): ToolbarState =>
  selectState(params).toolbar;

const selectActiveToolbarTab = (params: KeyedSelectorParams): ToolbarTab =>
  selectToolbar(params).activeTab;

export const useSelectActiveToolbarTab = createSelector(selectActiveToolbarTab);

const selectControlState = (params: KeyedSelectorParams): ControlState =>
  selectState(params).control;

export const useSelectControlState = createSelector(selectControlState);

const selectViewportMode = (params: KeyedSelectorParams): Viewport.Mode =>
  selectState(params).mode;

export const useSelectViewportMode = createSelector(selectViewportMode);

const selectHiddenLines = (params: KeyedSelectorParams): string[] =>
  selectState(params).hiddenLines;

export const useSelectHiddenLines = createSelector(selectHiddenLines);

const selectMeasureMode = (params: KeyedSelectorParams): lineplot.measure.Mode =>
  selectState(params).measure.mode;

export const useSelectMeasureMode = createSelector(selectMeasureMode);

const selectSelection = (params: KeyedSelectorParams): SelectionState =>
  selectState(params).selection;

export const useSelectSelection = createSelector(selectSelection);

const selectSelectedRules = (params: KeyedSelectorParams): string[] =>
  selectState(params).selectedRules;

export const useSelectSelectedRules = createSelector(selectSelectedRules);

const selectAnnotationsVisible = (params: KeyedSelectorParams): boolean =>
  selectState(params).annotations.visible;

export const useSelectAnnotationsVisible = createSelector(selectAnnotationsVisible);

const focusedKey = (
  pan: panel.Panel | undefined,
  tabKey: panel.TabKey | undefined,
): client.Key | undefined => {
  if (pan == null || tabKey == null) return undefined;
  const tab = panel.findTab(pan.root, tabKey);
  if (tab?.variant === "resource" && tab.resource.type === "lineplot")
    return tab.resource.key;
  return undefined;
};

/**
 * @returns a getter for the key of the line plot the user is looking at: the focused
 * tab of the window's selected panel, when that tab shows a line plot. Returns
 * undefined for every other focused tab, and while the panel is absent from the cache.
 */
export const useGetFocusedKey = (): (() => client.Key | undefined) => {
  const getSelectedPanel = Panel.useGetSelected();
  const getFocusedTabKey = Panel.useGetFocusedTab();
  const synnax = Synnax.use();
  return () => {
    const panelKey = getSelectedPanel();
    if (panelKey == null) return undefined;
    const cached = synnax?.panels.getCached(panelKey);
    return focusedKey(query.isLive(cached) ? cached : undefined, getFocusedTabKey());
  };
};

/**
 * @returns the key of the focused line plot, as {@link useGetFocusedKey}, re-rendering
 * when the selected panel, its focused tab, or the content of that tab changes.
 */
export const useSelectFocusedKey = (): client.Key | undefined => {
  const panelKey = Panel.useSelectSelected();
  const tabKey = Panel.useSelectFocusedTab();
  const { data } = PPanel.useResult(panelKey == null ? null : { key: panelKey });
  return focusedKey(data, tabKey);
};
