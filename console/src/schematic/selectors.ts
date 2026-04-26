// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Control } from "@synnaxlabs/pluto";

import { useMemoSelect } from "@/hooks";
import {
  type LegendState,
  type PendingUpload,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarTab,
  type Viewport,
  ZERO_STATE,
} from "@/schematic/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectOptional = (state: StoreState, key: string): State | undefined =>
  selectSliceState(state).schematics[key];

export const useSelectOptional = (key: string): State | undefined =>
  useMemoSelect((state: StoreState) => selectOptional(state, key), [key]);

export const selectSelected = (state: StoreState, key: string): string[] =>
  selectOptional(state, key)?.selected ?? [];

export const useSelectSelected = (key: string): string[] =>
  useMemoSelect((state: StoreState) => selectSelected(state, key), [key]);

export const selectControlStatus = (state: StoreState, key: string): Control.Status =>
  selectOptional(state, key)?.control ?? "released";

export const useSelectControlStatus = (key: string): Control.Status =>
  useMemoSelect((state: StoreState) => selectControlStatus(state, key), [key]);

export const selectActiveToolbarTab = (state: StoreState, key: string): ToolbarTab =>
  selectOptional(state, key)?.activeToolbarTab ?? "symbols";

export const useSelectActiveToolbarTab = (key: string): ToolbarTab =>
  useMemoSelect((state: StoreState) => selectActiveToolbarTab(state, key), [key]);

export const selectSelectedSymbolGroup = (state: StoreState, key: string): string =>
  selectOptional(state, key)?.selectedSymbolGroup ?? "general";

export const useSelectSelectedSymbolGroup = (key: string): string =>
  useMemoSelect((state: StoreState) => selectSelectedSymbolGroup(state, key), [key]);

export const selectLegend = (state: StoreState, key: string): LegendState =>
  selectOptional(state, key)?.legend ?? ZERO_STATE.legend;

export const useSelectLegend = (key: string): LegendState =>
  useMemoSelect((state: StoreState) => selectLegend(state, key), [key]);

export const selectLegendVisible = (state: StoreState, key: string): boolean =>
  selectOptional(state, key)?.legend.visible ?? false;

export const useSelectLegendVisible = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectLegendVisible(state, key), [key]);

export const selectEditable = (state: StoreState, key: string): boolean =>
  selectOptional(state, key)?.editable ?? true;

export const useSelectEditable = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectEditable(state, key), [key]);

export const selectFitViewOnResize = (state: StoreState, key: string): boolean =>
  selectOptional(state, key)?.fitViewOnResize ?? false;

export const useSelectFitViewOnResize = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectFitViewOnResize(state, key), [key]);

export const selectViewport = (state: StoreState, key: string): Viewport =>
  selectOptional(state, key)?.viewport ?? ZERO_STATE.viewport;

export const useSelectViewport = (key: string): Viewport =>
  useMemoSelect((state: StoreState) => selectViewport(state, key), [key]);

export const selectPendingUpload = (
  state: StoreState,
  key: string,
): PendingUpload | undefined => selectOptional(state, key)?.pendingUpload;

export const useSelectPendingUpload = (key: string): PendingUpload | undefined =>
  useMemoSelect((state: StoreState) => selectPendingUpload(state, key), [key]);
