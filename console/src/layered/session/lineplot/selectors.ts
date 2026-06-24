// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot as client } from "@synnaxlabs/client";
import { LinePlot, type Viewport } from "@synnaxlabs/pluto";
import { type lineplot } from "@synnaxlabs/pluto/ether";
import { type record } from "@synnaxlabs/x";

import { useMemoSelect } from "@/hooks";
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
} from "@/layered/session/lineplot/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export interface KeyedSelectorParams extends record.Keyed<client.Key> {
  state: StoreState;
}

const createSelector = <R>(selector: (params: KeyedSelectorParams) => R) =>
  LinePlot.Scope.bindHook(
    ({ key }: Omit<KeyedSelectorParams, "state">): R =>
      useMemoSelect((state: StoreState) => selector({ state, key }), [key]),
  );

export const selectState = ({ state, key }: KeyedSelectorParams): State =>
  selectSliceState(state).plots[key] ?? ZERO_STATE;

export const useSelect = createSelector(selectState);

export const selectToolbar = (params: KeyedSelectorParams): ToolbarState =>
  selectState(params).toolbar;

export const useSelectToolbar = createSelector(selectToolbar);

export const selectActiveToolbarTab = (params: KeyedSelectorParams): ToolbarTab =>
  selectToolbar(params).activeTab;

export const useSelectActiveToolbarTab = createSelector(selectActiveToolbarTab);

export const selectControlState = (params: KeyedSelectorParams): ControlState =>
  selectState(params).control;

export const useSelectControlState = createSelector(selectControlState);

export const selectViewportMode = (params: KeyedSelectorParams): Viewport.Mode =>
  selectState(params).mode;

export const useSelectViewportMode = createSelector(selectViewportMode);

export const selectHiddenLines = (params: KeyedSelectorParams): string[] =>
  selectState(params).hiddenLines;

export const useSelectHiddenLines = createSelector(selectHiddenLines);

export const selectMeasureMode = (params: KeyedSelectorParams): lineplot.measure.Mode =>
  selectState(params).measure.mode;

export const useSelectMeasureMode = createSelector(selectMeasureMode);

export const selectSelection = (params: KeyedSelectorParams): SelectionState =>
  selectState(params).selection;

export const useSelectSelection = createSelector(selectSelection);

export const selectSelectedRules = (params: KeyedSelectorParams): string[] =>
  selectState(params).selectedRules;

export const useSelectSelectedRules = createSelector(selectSelectedRules);

export const selectAnnotationsVisible = (params: KeyedSelectorParams): boolean =>
  selectState(params).annotations.visible;

export const useSelectAnnotationsVisible = createSelector(selectAnnotationsVisible);
