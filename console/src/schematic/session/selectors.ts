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
import { type control } from "@synnaxlabs/x";

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
} from "@/schematic/session/slice";

export const createSelector =
  <S extends object, R>(
    selector: (state: S, key: schematic.Key) => R,
  ): ((override?: schematic.Key) => R) =>
  (override) => {
    const key = Schematic.useKey(override);
    return useMemoSelect((state: S) => selector(state, key), [key]);
  };

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).schematics[key] ?? ZERO_STATE;

export const useSelect = createSelector(select);

export const selectSelected = (state: StoreState, key: string): string[] =>
  select(state, key)?.selected;

export const useSelectSelected = createSelector(selectSelected);

export const selectControlStatus = (state: StoreState, key: string): Control.Status =>
  select(state, key).control.status;

export const useSelectControlStatus = createSelector(selectControlStatus);

export const selectControlIsAcquired = (state: StoreState, key: string): boolean =>
  selectControlStatus(state, key) === "acquired";

export const useSelectControlIsAcquired = createSelector(selectControlIsAcquired);

export const selectAuthority = (state: StoreState, key: string): control.Authority =>
  select(state, key).control.authority;

export const useSelectAuthority = createSelector(selectAuthority);

export const selectActiveToolbarTab = (state: StoreState, key: string): ToolbarTab =>
  selectToolbar(state, key).selectedTab;

export const useSelectActiveToolbarTab = createSelector(selectActiveToolbarTab);

export const selectToolbar = (state: StoreState, key: string): ToolbarState =>
  select(state, key).toolbar;

export const useSelectToolbar = createSelector(selectToolbar);

export const selectSelectedSymbolGroup = (state: StoreState, key: string): string =>
  selectToolbar(state, key).selectedSymbolGroup;

export const useSelectSelectedSymbolGroup = createSelector(selectSelectedSymbolGroup);

export const selectLegend = (state: StoreState, key: string): LegendState =>
  select(state, key).legend;

export const useSelectLegend = createSelector(selectLegend);

export const selectLegendVisible = (state: StoreState, key: string): boolean =>
  select(state, key).legend.visible;

export const useSelectLegendVisible = createSelector(selectLegendVisible);

export const selectEditable = (state: StoreState, key: string): boolean =>
  select(state, key).editable;

const useSelectEditableBase = createSelector(selectEditable);

export const selectFitViewOnResize = (state: StoreState, key: string): boolean =>
  select(state, key).fitViewOnResize;

export const useSelectFitViewOnResize = createSelector(selectFitViewOnResize);

export const selectViewport = (state: StoreState, key: string): Viewport =>
  select(state, key).viewport;

export const useSelectViewport = createSelector(selectViewport);

export interface UseSelectEditableReturn {
  isCurrentlyEditable: boolean;
  canEdit: boolean;
}

export const useSelectEditable = (
  overrideKey?: schematic.Key,
): UseSelectEditableReturn => {
  const key = Schematic.useKey(overrideKey);
  const isSnapshot = Schematic.useSelectSnapshot({});
  const hasUpdatePermission = Access.useUpdateGranted(schematic.ontologyID(key));
  const editable = useSelectEditableBase();
  const canEdit = hasUpdatePermission && !isSnapshot;
  const isCurrentlyEditable = canEdit && editable;
  return { canEdit, isCurrentlyEditable };
};
