// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UseViewportTriggers, Viewport, ViewportMode } from "@synnaxlabs/pluto";

import { useMemoSelect } from "@/hooks";
import {
  LinePlotState,
  LineSliceState,
  LineStoreState,
  LINE_SLICE_NAME,
  LineToolbarState,
} from "@/line/store/slice";
import { XAxisRecord } from "@/vis";
import { Range, WorkspaceStoreState, selectRanges } from "@/workspace";

export const selectLineSliceState = (state: LineStoreState): LineSliceState =>
  state[LINE_SLICE_NAME];

export const selectLinePlot = (state: LineStoreState, key: string): LinePlotState =>
  selectLineSliceState(state).plots[key];

export const useSelectLinePlot = (key: string): LinePlotState =>
  useMemoSelect((state: LineStoreState) => selectLinePlot(state, key), [key]);

export const useSelectLinePlotRanges = (key: string): XAxisRecord<Range[]> => {
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

export const selectLineViewportMode = (state: LineStoreState): ViewportMode =>
  selectLineSliceState(state).mode;

export const useSelectLineViewportMode = (): ViewportMode =>
  useMemoSelect(selectLineViewportMode, []);

export const selecLineViewportTriggers = (
  state: LineStoreState
): UseViewportTriggers => {
  return Viewport.DEFAULT_TRIGGERS[selectLineViewportMode(state)];
};

export const useSelectLineViewportTriggers = (): UseViewportTriggers =>
  useMemoSelect(selecLineViewportTriggers, []);
