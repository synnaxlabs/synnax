// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Viewport } from "@synnaxlabs/pluto";
import { type measure } from "@synnaxlabs/pluto/ether";
import { type bounds } from "@synnaxlabs/x";

import { useMemoSelect } from "@/hooks";
import { type AxisKey, type XAxisRecord } from "@/lineplot/axis";
import {
  type ControlState,
  type LineState,
  type RuleState,
  type SelectionState,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarState,
} from "@/lineplot/slice";
import { Range } from "@/range";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).plots[key];

export const selectOptional = (state: StoreState, key: string): State | undefined =>
  selectSliceState(state).plots[key];

export const selectMultiple = (state: StoreState, keys: string[]): State[] =>
  keys.map((key) => select(state, key));

export const useSelect = (key: string): State =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const selectRanges = (
  state: StoreState & Range.StoreState,
  key: string,
): XAxisRecord<Range.Range[]> => {
  const ranges = select(state, key).ranges;
  return {
    x1: Range.selectMultiple(state, ranges.x1),
    x2: Range.selectMultiple(state, ranges.x2),
  };
};

export const useSelectRanges = (key: string): XAxisRecord<Range.Range[]> =>
  useMemoSelect(
    (state: StoreState & Range.StoreState) => selectRanges(state, key),
    [key],
  );

export const selectToolbar = (state: StoreState, key: string): ToolbarState =>
  select(state, key).toolbar;

export const useSelectToolbar = (key: string): ToolbarState =>
  useMemoSelect((state: StoreState) => selectToolbar(state, key), [key]);

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

export const selectMeasureMode = (state: StoreState, key: string): measure.Mode =>
  select(state, key).measure.mode;

export const useSelectMeasureMode = (key: string): measure.Mode =>
  useMemoSelect((state: StoreState) => selectMeasureMode(state, key), [key]);

export const selectSelection = (state: StoreState, key: string): SelectionState =>
  select(state, key).selection;

export const useSelectSelection = (key: string): SelectionState =>
  useMemoSelect((state: StoreState) => selectSelection(state, key), [key]);

export const selectAxes = (state: StoreState, key: string) =>
  select(state, key).axes.axes;

export const useSelectAxes = (key: string) =>
  useMemoSelect((state: StoreState) => selectAxes(state, key), [key]);

export const selectAxisBounds = (
  state: StoreState,
  key: string,
  axisKey: AxisKey,
): bounds.Bounds => {
  const p = select(state, key);
  return p.axes.axes[axisKey].bounds;
};

export const useSelectAxisBounds = (key: string, axisKey: AxisKey): bounds.Bounds =>
  useMemoSelect(
    (state: StoreState) => selectAxisBounds(state, key, axisKey),
    [key, axisKey],
  );

export const selectVersion = (state: StoreState, key: string): string | undefined =>
  selectOptional(state, key)?.version;

export const useSelectVersion = (key: string): string | undefined =>
  useMemoSelect((state: StoreState) => selectVersion(state, key), [key]);

export const selectRules = (state: StoreState, key: string): RuleState[] =>
  select(state, key).rules;

export const useSelectRules = (key: string): RuleState[] =>
  useMemoSelect((state: StoreState) => selectRules(state, key), [key]);

export const selectRule = (
  state: StoreState,
  key: string,
  ruleKey?: string,
): RuleState | undefined => {
  if (ruleKey == null) return undefined;
  return select(state, key).rules.find(({ key: k }) => k === ruleKey);
};

export const useSelectRule = (
  layoutKey: string,
  ruleKey?: string,
): RuleState | undefined =>
  useMemoSelect(
    (state: StoreState) => selectRule(state, layoutKey, ruleKey),
    [layoutKey, ruleKey],
  );

export const selectLines = (state: StoreState, key: string): LineState[] =>
  select(state, key).lines;

export const useSelectLines = (key: string): LineState[] =>
  useMemoSelect((state: StoreState) => selectLines(state, key), [key]);

export const selectLineKeys = (state: StoreState, key: string): string[] =>
  select(state, key).lines.map(({ key }) => key);

export const useSelectLineKeys = (key: string): string[] =>
  useMemoSelect((state: StoreState) => selectLineKeys(state, key), [key]);

export const selectLine = (
  state: StoreState,
  plotKey: string,
  lineKey?: string,
): LineState | undefined => {
  if (lineKey == null) return undefined;
  return select(state, plotKey).lines.find(({ key: k }) => k === lineKey);
};

export const useSelectLine = (
  plotKey: string,
  lineKey?: string,
): LineState | undefined =>
  useMemoSelect(
    (state: StoreState) => selectLine(state, plotKey, lineKey),
    [plotKey, lineKey],
  );
