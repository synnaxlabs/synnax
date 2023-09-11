// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Viewport } from "@synnaxlabs/pluto";

import { useMemoSelect } from "@/hooks";
import {
  type State,
  type SliceState,
  type StoreState,
  SLICE_NAME,
  type ToolbarState,
  type ControlState,
} from "@/lineplot/slice";
import { type Vis } from "@/vis";
import { Workspace } from "@/workspace";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).plots[key];

export const useSelect = (key: string): State =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const selectRanges = (key: string): Vis.XAxisRecord<Workspace.Range[]> => {
  return useMemoSelect(
    (state: StoreState & Workspace.StoreState) => {
      const p = select(state, key);
      return {
        x1: Workspace.selectRanges(state, p.ranges.x1),
        x2: Workspace.selectRanges(state, p.ranges.x2),
      };
    },
    [key]
  );
};

export const selectToolbar = (state: StoreState): ToolbarState =>
  selectSliceState(state).toolbar;

export const useSelectToolbar = (): ToolbarState => useMemoSelect(selectToolbar, []);

export const selectControlState = (state: StoreState): ControlState =>
  selectSliceState(state).control;

export const useSelectControlState = (): ControlState =>
  useMemoSelect(selectControlState, []);

export const selectViewportMode = (state: StoreState): Viewport.Mode =>
  selectSliceState(state).mode;

export const useSelectViewportMode = (): Viewport.Mode =>
  useMemoSelect(selectViewportMode, []);
