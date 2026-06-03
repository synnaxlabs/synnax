// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Viewport } from "@synnaxlabs/pluto";
import { type lineplot } from "@synnaxlabs/pluto/ether";

import { useMemoSelect } from "@/hooks";
import {
  type ControlState,
  type PendingUpload,
  type SelectionState,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarState,
  type ToolbarTab,
} from "@/lineplot/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).plots[key];

export const selectOptional = (state: StoreState, key: string): State | undefined =>
  selectSliceState(state).plots[key];

export const useSelect = (key: string): State =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const useSelectOptional = (key: string): State | undefined =>
  useMemoSelect((state: StoreState) => selectOptional(state, key), [key]);

export const selectExists = (state: StoreState, key: string): boolean =>
  selectOptional(state, key) != null;

export const useSelectExists = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectExists(state, key), [key]);

export const selectIsRemoteCreated = (
  state: StoreState,
  key: string,
): boolean | undefined => selectOptional(state, key)?.remoteCreated;

export const useSelectIsRemoteCreated = (key: string): boolean | undefined =>
  useMemoSelect((state: StoreState) => selectIsRemoteCreated(state, key), [key]);

export const selectPendingUpload = (
  state: StoreState,
  key: string,
): PendingUpload | undefined => selectOptional(state, key)?.pendingUpload;

export const useSelectPendingUpload = (key: string): PendingUpload | undefined =>
  useMemoSelect((state: StoreState) => selectPendingUpload(state, key), [key]);

export const selectToolbar = (
  state: StoreState,
  key: string,
): ToolbarState | undefined => selectOptional(state, key)?.toolbar;

export const useSelectToolbar = (key: string): ToolbarState | undefined =>
  useMemoSelect((state: StoreState) => selectToolbar(state, key), [key]);

export const selectActiveToolbarTab = (state: StoreState, key: string): ToolbarTab =>
  select(state, key).toolbar.activeTab;

export const useSelectActiveToolbarTab = (key: string): ToolbarTab =>
  useMemoSelect((state: StoreState) => selectActiveToolbarTab(state, key), [key]);

export const selectControlState = (state: StoreState, key: string): ControlState =>
  select(state, key).control;

export const useSelectControlState = (key: string): ControlState =>
  useMemoSelect((state: StoreState) => selectControlState(state, key), [key]);

export const selectControlStateOptional = (
  state: StoreState,
  key: string,
): ControlState | undefined => selectOptional(state, key)?.control;

export const useSelectControlStateOptional = (key: string): ControlState | undefined =>
  useMemoSelect((state: StoreState) => selectControlStateOptional(state, key), [key]);

export const selectViewportMode = (state: StoreState, key: string): Viewport.Mode =>
  select(state, key).mode;

export const useSelectViewportMode = (key: string): Viewport.Mode =>
  useMemoSelect((state: StoreState) => selectViewportMode(state, key), [key]);

export const selectMeasureMode = (
  state: StoreState,
  key: string,
): lineplot.measure.Mode => select(state, key).measure.mode;

export const useSelectMeasureMode = (key: string): lineplot.measure.Mode =>
  useMemoSelect((state: StoreState) => selectMeasureMode(state, key), [key]);

export const selectSelection = (state: StoreState, key: string): SelectionState =>
  select(state, key).selection;

export const useSelectSelection = (key: string): SelectionState =>
  useMemoSelect((state: StoreState) => selectSelection(state, key), [key]);

export const selectVersion = (state: StoreState, key: string): string | undefined =>
  selectOptional(state, key)?.version;

export const useSelectVersion = (key: string): string | undefined =>
  useMemoSelect((state: StoreState) => selectVersion(state, key), [key]);

export const selectSelectedRules = (state: StoreState, key: string): string[] =>
  select(state, key).selectedRules;

export const useSelectSelectedRules = (key: string): string[] =>
  useMemoSelect((state: StoreState) => selectSelectedRules(state, key), [key]);
