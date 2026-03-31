// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { type Control } from "@synnaxlabs/pluto";

import { useMemoSelect } from "@/hooks";
import {
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarTab,
} from "@/schematic/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectRequired = (state: StoreState, key: string): State => {
  const schematic = selectSliceState(state).schematics[key];
  if (schematic == null)
    throw new UnexpectedError(`Schematic not found for key: ${key}`);
  return schematic;
};

export const useSelectRequired = (key: string): State =>
  useMemoSelect((state: StoreState) => selectRequired(state, key), [key]);

export const selectOptional = (state: StoreState, key: string): State | undefined =>
  selectSliceState(state).schematics[key];

export const useSelectOptional = (key: string): State | undefined =>
  useMemoSelect((state: StoreState) => selectOptional(state, key), [key]);

export const selectSelected = (state: StoreState, key: string): string[] =>
  selectOptional(state, key)?.selected ?? [];

export const useSelectSelected = (key: string): string[] =>
  useMemoSelect((state: StoreState) => selectSelected(state, key), [key]);

export const selectControlStatus = (
  state: StoreState,
  layoutKey: string,
): Control.Status | undefined => selectOptional(state, layoutKey)?.control;

export const useSelectControlStatus = (layoutKey: string): Control.Status | undefined =>
  useMemoSelect(
    (state: StoreState) => selectControlStatus(state, layoutKey),
    [layoutKey],
  );

export const selectActiveToolbarTab = (
  state: StoreState,
  key: string,
): ToolbarTab | undefined => selectOptional(state, key)?.activeToolbarTab;

export const useSelectActiveToolbarTab = (key: string): ToolbarTab | undefined =>
  useMemoSelect((state: StoreState) => selectActiveToolbarTab(state, key), [key]);

export const selectSelectedSymbolGroup = (state: StoreState, key: string): string =>
  selectOptional(state, key)?.selectedSymbolGroup ?? "general";

export const useSelectSelectedSymbolGroup = (key: string): string =>
  useMemoSelect((state: StoreState) => selectSelectedSymbolGroup(state, key), [key]);

export const selectLegendVisible = (
  state: StoreState,
  key: string,
): boolean | undefined => selectOptional(state, key)?.legend.visible;

export const useSelectLegendVisible = (key: string): boolean | undefined =>
  useMemoSelect((state: StoreState) => selectLegendVisible(state, key), [key]);

export const selectEditable = (state: StoreState, key: string): boolean =>
  selectOptional(state, key)?.editable ?? true;

export const useSelectEditable = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectEditable(state, key), [key]);

export const selectFitViewOnResize = (state: StoreState, key: string): boolean =>
  selectOptional(state, key)?.fitViewOnResize ?? false;

export const useSelectFitViewOnResize = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectFitViewOnResize(state, key), [key]);
