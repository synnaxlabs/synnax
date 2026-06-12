// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Control } from "@synnaxlabs/pluto";
import { type control } from "@synnaxlabs/x";

import { useMemoSelect } from "@/hooks";
import {
  type LegendState,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarTab,
  type Viewport,
} from "@/schematic/slice";
import { type ToolbarState, ZERO_STATE } from "@/schematic/types";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectSelected = (state: StoreState, key: string): string[] =>
  select(state, key)?.selected ?? [];

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).schematics[key];

export const selectOptional = select as (
  state: StoreState,
  key: string,
) => State | undefined;

export const useSelect = (key: string): State =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const useSelectOptional = (key: string): State | undefined =>
  useMemoSelect((state: StoreState) => selectOptional(state, key), [key]);

export const selectExists = (state: StoreState, key: string): boolean =>
  selectOptional(state, key) != null;

export const useSelectExists = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectExists(state, key), [key]);

export const useSelectSelected = (key: string): string[] =>
  useMemoSelect((state: StoreState) => selectSelected(state, key), [key]);

export const selectControlStatus = (state: StoreState, key: string): Control.Status =>
  select(state, key)?.controlStatus ?? "released";

export const useSelectControlStatus = (key: string): Control.Status =>
  useMemoSelect((state: StoreState) => selectControlStatus(state, key), [key]);

export const selectAuthority = (state: StoreState, key: string): control.Authority =>
  select(state, key)?.authority ?? 1;

export const useSelectAuthority = (key: string): control.Authority =>
  useMemoSelect((state: StoreState) => selectAuthority(state, key), [key]);

export const selectActiveToolbarTab = (state: StoreState, key: string): ToolbarTab =>
  selectToolbar(state, key).activeTab;

export const useSelectActiveToolbarTab = (key: string): ToolbarTab =>
  useMemoSelect((state: StoreState) => selectActiveToolbarTab(state, key), [key]);

export const selectToolbar = (state: StoreState, key: string): ToolbarState =>
  select(state, key).toolbar;

export const useSelectToolbar = (key: string): ToolbarState =>
  useMemoSelect((state: StoreState) => selectToolbar(state, key), [key]);

export const selectSelectedSymbolGroup = (state: StoreState, key: string): string =>
  selectToolbar(state, key).selectedSymbolGroup;

export const useSelectSelectedSymbolGroup = (key: string): string =>
  useMemoSelect((state: StoreState) => selectSelectedSymbolGroup(state, key), [key]);

export const selectLegend = (state: StoreState, key: string): LegendState =>
  select(state, key)?.legend ?? ZERO_STATE.legend;

export const useSelectLegend = (key: string): LegendState =>
  useMemoSelect((state: StoreState) => selectLegend(state, key), [key]);

export const selectLegendVisible = (state: StoreState, key: string): boolean =>
  select(state, key).legend.visible;

export const useSelectLegendVisible = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectLegendVisible(state, key), [key]);

export const selectEditable = (state: StoreState, key: string): boolean =>
  select(state, key).editable;

export const useSelectEditable = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectEditable(state, key), [key]);

export const selectFitViewOnResize = (state: StoreState, key: string): boolean =>
  select(state, key).fitViewOnResize;

export const useSelectFitViewOnResize = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectFitViewOnResize(state, key), [key]);

export const selectViewport = (state: StoreState, key: string): Viewport =>
  select(state, key).viewport;

export const useSelectViewport = (key: string): Viewport =>
  useMemoSelect((state: StoreState) => selectViewport(state, key), [key]);
