// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemoSelect } from "@/hooks";
import {
  LinePlotState,
  SliceState,
  LineStoreState,
  SLICE_NAME,
  LineToolbarState,
  LineControlState,
} from "@/line/slice";
import { Vis } from "@/vis";
import { Range, WorkspaceStoreState, selectRanges } from "@/workspace";

export const selectLineSliceState = (state: LineStoreState): SliceState =>
  state[SLICE_NAME];

export const selectLinePlot = (state: LineStoreState, key: string): LinePlotState =>
  selectLineSliceState(state).plots[key];

export const useSelectLinePlot = (key: string): LinePlotState =>
  useMemoSelect((state: LineStoreState) => selectLinePlot(state, key), [key]);

export const useSelectLinePlotRanges = (key: string): Vis.XAxisRecord<Range[]> => {
  return useMemoSelect(
    (state: LineStoreState & WorkspaceStoreState) => {
      const p = selectLinePlot(state, key);
      return {
        x1: selectRanges(state, p.ranges.x1),
        x2: selectRanges(state, p.ranges.x2),
      };
    },
    [key]
  );
};

export const selectLineToolbar = (state: LineStoreState): LineToolbarState =>
  selectLineSliceState(state).toolbar;

export const useSelectLineToolbar = (): LineToolbarState =>
  useMemoSelect(selectLineToolbar, []);

export const selectLineControlState = (state: LineStoreState): LineControlState =>
  selectLineSliceState(state).control;

export const useSelectLineControlState = (): LineControlState =>
  useMemoSelect(selectLineControlState, []);
