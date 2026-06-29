// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Access, Arc, type Diagram, type Viewport } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

import { useMemoSelect } from "@/hooks";
import {
  type GraphState,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarState,
  ZERO_STATE,
} from "@/layered/session/arc/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export interface KeyedSelectorParams extends record.Keyed<arc.Key> {
  state: StoreState;
}

const createSelector = <R>(selector: (params: KeyedSelectorParams) => R) =>
  Arc.Scope.bindHook(
    ({ key }: Omit<KeyedSelectorParams, "state">): R =>
      useMemoSelect((state: StoreState) => selector({ state, key }), [key]),
  );

export const selectState = ({ state, key }: KeyedSelectorParams): State =>
  selectSliceState(state).arcs[key] ?? ZERO_STATE;

export const selectGraph = (params: KeyedSelectorParams): GraphState =>
  selectState(params).graph;

export const selectSelected = (params: KeyedSelectorParams): string[] =>
  selectGraph(params).selected;

export const selectEditable = (params: KeyedSelectorParams): boolean =>
  selectGraph(params).editable;

export const selectViewport = (params: KeyedSelectorParams): Diagram.Viewport =>
  selectGraph(params).viewport;

export const selectToolbar = (params: KeyedSelectorParams): ToolbarState =>
  selectState(params).toolbar;

export const selectViewportMode = (state: KeyedSelectorParams): Viewport.Mode =>
  selectGraph(state).viewport.mode;

const selectFitViewOnResize = (state: KeyedSelectorParams): boolean =>
  selectGraph(state).fitViewOnResize;

export const useSelect = createSelector(selectState);
export const useSelectSelected = createSelector(selectSelected);
export const useSelectViewport = createSelector(selectViewport);
export const useSelectViewportMode = createSelector(selectViewportMode);
export const useSelectToolbar = createSelector(selectToolbar);
export const useSelectFitViewOnResize = createSelector(selectFitViewOnResize);

export interface UseSelectEditableReturn {
  isCurrentlyEditable: boolean;
  canEdit: boolean;
}

const useSelectEditableBase = createSelector(selectEditable);

export const useSelectEditable = (args?: {
  key?: arc.Key;
}): UseSelectEditableReturn => {
  const key = Arc.useKey(args?.key);
  const hasUpdatePermission = Access.useUpdateGranted(arc.ontologyID(key));
  const editable = useSelectEditableBase(args);
  const canEdit = hasUpdatePermission;
  const isCurrentlyEditable = hasUpdatePermission && editable;
  return { canEdit, isCurrentlyEditable };
};
