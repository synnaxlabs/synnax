// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { lineplot as client } from "@synnaxlabs/client";
import { LinePlot, type Viewport } from "@synnaxlabs/pluto";
import type { lineplot } from "@synnaxlabs/pluto/ether";
import type { record } from "@synnaxlabs/x";

import {
  type ControlState,
  type Hidden,
  type SelectionState,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarState,
  type ToolbarTab,
  ZERO_STATE,
} from "@/session/lineplot/slice";
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

const selectHidden = (params: KeyedSelectorParams): Hidden => {
  const { hiddenLines, hiddenRanges, hiddenTraces } = selectState(params);
  return { lines: hiddenLines, ranges: hiddenRanges, traces: hiddenTraces };
};

/** Selects the items of the plot that do not show. */
export const useSelectHidden = createSelector(selectHidden);

const selectSoloRestore = (params: KeyedSelectorParams): Hidden | null =>
  selectState(params).soloRestore;

export const useSelectSoloRestore = createSelector(selectSoloRestore);

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
