// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Access, type Control, Schematic } from "@synnaxlabs/pluto";
import { type control, type record } from "@synnaxlabs/x";

import { useMemoSelect } from "@/hooks";
import {
  type LegendState,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarState,
  type ToolbarTab,
  type Viewport,
  ZERO_STATE,
} from "@/layered/session/schematic/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export interface KeyedSelectorParams extends record.Keyed<schematic.Key> {
  state: StoreState;
}

const createSelector = <R>(selector: (params: KeyedSelectorParams) => R) =>
  Schematic.Scope.bindHook(
    ({ key }: Omit<KeyedSelectorParams, "state">): R =>
      useMemoSelect((state: StoreState) => selector({ state, key }), [key]),
  );

export const selectState = ({ state, key }: KeyedSelectorParams): State =>
  selectSliceState(state).schematics[key] ?? ZERO_STATE;

export const useSelect = createSelector(selectState);

export const selectSelected = (params: KeyedSelectorParams): string[] =>
  selectState(params)?.selected;

export const useSelectSelected = createSelector(selectSelected);

export const selectControlStatus = (params: KeyedSelectorParams): Control.Status =>
  selectState(params).control.status;

export const useSelectControlStatus = createSelector(selectControlStatus);

export const selectControlIsAcquired = (params: KeyedSelectorParams): boolean =>
  selectControlStatus(params) === "acquired";

export const useSelectControlIsAcquired = createSelector(selectControlIsAcquired);

export const selectAuthority = (params: KeyedSelectorParams): control.Authority =>
  selectState(params).control.authority;

export const useSelectAuthority = createSelector(selectAuthority);

export const selectActiveToolbarTab = (params: KeyedSelectorParams): ToolbarTab =>
  selectToolbar(params).selectedTab;

export const useSelectActiveToolbarTab = createSelector(selectActiveToolbarTab);

export const selectToolbar = (params: KeyedSelectorParams): ToolbarState =>
  selectState(params).toolbar;

export const useSelectToolbar = createSelector(selectToolbar);

export const selectSelectedSymbolGroup = (params: KeyedSelectorParams): string =>
  selectToolbar(params).selectedSymbolGroup;

export const useSelectSelectedSymbolGroup = createSelector(selectSelectedSymbolGroup);

export const selectLegend = (params: KeyedSelectorParams): LegendState =>
  selectState(params).legend;

export const useSelectLegend = createSelector(selectLegend);

export const selectLegendVisible = (params: KeyedSelectorParams): boolean =>
  selectState(params).legend.visible;

export const useSelectLegendVisible = createSelector(selectLegendVisible);

export const selectEditable = (params: KeyedSelectorParams): boolean =>
  selectState(params).editable;

const useSelectEditableBase = createSelector(selectEditable);

export const selectFitViewOnResize = (params: KeyedSelectorParams): boolean =>
  selectState(params).fitViewOnResize;

export const useSelectFitViewOnResize = createSelector(selectFitViewOnResize);

export const selectViewport = (params: KeyedSelectorParams): Viewport =>
  selectState(params).viewport;

export const useSelectViewport = createSelector(selectViewport);

export interface UseSelectEditableReturn {
  isCurrentlyEditable: boolean;
  canEdit: boolean;
}

export const useSelectEditable = (args?: {
  key?: schematic.Key;
}): UseSelectEditableReturn => {
  const key = Schematic.useKey(args?.key);
  const isSnapshot = Schematic.useSelectSnapshot();
  const hasUpdatePermission = Access.useUpdateGranted(schematic.ontologyID(key));
  const editable = useSelectEditableBase(args);
  const canEdit = hasUpdatePermission && !isSnapshot;
  const isCurrentlyEditable = canEdit && editable;
  return { canEdit, isCurrentlyEditable };
};
