// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Diagram, type Viewport } from "@synnaxlabs/pluto";

import {
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarState,
} from "@/arc/slice";
import { type Mode } from "@/arc/types";
import { useMemoSelect } from "@/hooks";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).arcs[key];

export const selectOptional = select as (
  state: StoreState,
  key: string,
) => State | undefined;

export const useSelect = (key: string): State =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const useSelectOptional = useSelect as (key: string) => State | undefined;

export const selectMany = (state: StoreState, keys: string[]): State[] =>
  keys.map((key) => select(state, key));

export const useSelectMany = (keys: string[]): State[] =>
  useMemoSelect((state: StoreState) => selectMany(state, keys), [keys]);

export const selectSelected = (state: StoreState, layoutKey: string): string[] =>
  select(state, layoutKey).graph.selected;

export const useSelectSelected = (layoutKey: string): string[] =>
  useMemoSelect((state: StoreState) => selectSelected(state, layoutKey), [layoutKey]);

export const selectToolbar = (state: StoreState): ToolbarState =>
  selectSliceState(state).toolbar;

export const useSelectToolbar = (): ToolbarState => useMemoSelect(selectToolbar, []);

export const selectEditable = (state: StoreState, key: string): boolean =>
  select(state, key).graph.editable;

export const useSelectEditable = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectEditable(state, key), [key]);

export const selectViewportMode = (state: StoreState): Viewport.Mode =>
  selectSliceState(state).mode;

export const useSelectViewportMode = (): Viewport.Mode =>
  useMemoSelect(selectViewportMode, []);

export const selectViewport = (state: StoreState, key: string): Diagram.Viewport =>
  select(state, key).graph.viewport;

export const useSelectViewport = (key: string): Diagram.Viewport =>
  useMemoSelect((state: StoreState) => selectViewport(state, key), [key]);

export const selectVersion = (state: StoreState, key: string): string | undefined =>
  selectOptional(state, key)?.version;

export const useSelectVersion = (key: string): string | undefined =>
  useMemoSelect((state: StoreState) => selectVersion(state, key), [key]);

export const selectMode = (state: StoreState, key: string): Mode =>
  select(state, key).mode;

export const useSelectMode = (key: string): Mode =>
  useMemoSelect((state: StoreState) => selectMode(state, key), [key]);

export const selectOptionalMode = (state: StoreState, key: string): Mode =>
  selectOptional(state, key)?.mode;

export const useSelectOptionalMode = (key: string): Mode =>
  useMemoSelect((state: StoreState) => selectOptionalMode(state, key), [key]);

export const selectExists = (state: StoreState, key: string): boolean =>
  selectOptional(state, key) != null;

export const useSelectExists = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectExists(state, key), [key]);
