// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/lineplot/types/v0";
import * as v1 from "@/lineplot/types/v1";
import * as v2 from "@/lineplot/types/v2";
import * as v3 from "@/lineplot/types/v3";
import * as v4 from "@/lineplot/types/v4";
import * as v5 from "@/lineplot/types/v5";

export const titleStateZ = v5.titleStateZ;
export type TitleState = v5.TitleState;
export const ZERO_TITLE_STATE = v5.ZERO_TITLE_STATE;

export const legendStateZ = v5.legendStateZ;
export type LegendState = v5.LegendState;
export const ZERO_LEGEND_STATE = v5.ZERO_LEGEND_STATE;

export const viewportStateZ = v0.viewportStateZ;
export type ViewportState = v0.ViewportState;
export const ZERO_VIEWPORT_STATE = v0.ZERO_VIEWPORT_STATE;

export const selectionStateZ = v0.selectionStateZ;
export type SelectionState = v0.SelectionState;
export const ZERO_SELECTION_STATE = v0.ZERO_SELECTION_STATE;

export const axisStateZ = v5.axisStateZ;
export type AxisState = v5.AxisState;
export const ZERO_AXIS_STATE = v5.ZERO_AXIS_STATE;

export const axesStateZ = v5.axesStateZ;
export type AxesState = v5.AxesState;
export const ZERO_AXES_STATE = v5.ZERO_AXES_STATE;

export const lineStateZ = v5.lineStateZ;
export type LineState = v5.LineState;
export const ZERO_LINE_STATE = v5.ZERO_LINE_STATE;

export const linesStateZ = v5.linesStateZ;
export type LinesState = v5.LinesState;
export const ZERO_LINES_STATE = v5.ZERO_LINES_STATE;

export const ruleStateZ = v5.ruleStateZ;
export type RuleState = v5.RuleState;
export const ZERO_RULE_STATE = v5.ZERO_RULE_STATE;
export const DEFAULT_RULE_COLOR = v5.DEFAULT_RULE_COLOR;

export const rulesStateZ = v5.rulesStateZ;
export type RulesState = v5.RulesState;
export const ZERO_RULES_STATE = v5.ZERO_RULES_STATE;

export const channelsStateZ = v5.channelsStateZ;
export type ChannelsState = v5.ChannelsState;
export const ZERO_CHANNELS_STATE = v5.ZERO_CHANNELS_STATE;

export const rangesStateZ = v5.rangesStateZ;
export type RangesState = v5.RangesState;
export const ZERO_RANGES_STATE = v5.ZERO_RANGES_STATE;

export const annotationsStateZ = v4.annotationsStateZ;
export type AnnotationsState = v4.AnnotationsState;
export const ZERO_ANNOTATIONS_STATE = v4.ZERO_ANNOTATIONS_STATE;

export const stateZ = v5.stateZ;
export type State = v5.State;
export const ZERO_STATE = v5.ZERO_STATE;

export const toolbarTabZ = v0.toolbarTabZ;
export type ToolbarTab = v0.ToolbarTab;

export const toolbarStateZ = v0.toolbarStateZ;
export type ToolbarState = v0.ToolbarState;
export const ZERO_TOOLBAR_STATE = v0.ZERO_TOOLBAR_STATE;

export const clickModeZ = v0.clickModeZ;
export type ClickMode = v0.ClickMode;

export const controlStateZ = v0.controlStateZ;
export type ControlState = v0.ControlState;
export const ZERO_CONTROL_STATE = v0.ZERO_CONTROL_SATE;

export const sliceStateZ = v5.sliceStateZ;
export type SliceState = v5.SliceState;
export const ZERO_SLICE_STATE = v5.ZERO_SLICE_STATE;

// Re-export typed body helpers for server projection.
export const toWire = v5.toWire;
export const fromWire = v5.fromWire;

export type AnyState = v0.State | v1.State | v2.State | v3.State | v4.State | v5.State;
export type AnySliceState =
  | v0.SliceState
  | v1.SliceState
  | v2.SliceState
  | v3.SliceState
  | v4.SliceState
  | v5.SliceState;

export const STATE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.stateMigration,
  [v1.VERSION]: v2.stateMigration,
  [v2.VERSION]: v3.stateMigration,
  [v3.VERSION]: v4.stateMigration,
  [v4.VERSION]: v5.stateMigration,
};

export const migrateState = migrate.migrator<AnyState, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrations: STATE_MIGRATIONS,
  def: ZERO_STATE,
});

export const SLICE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.sliceMigration,
  [v1.VERSION]: v2.sliceMigration,
  [v2.VERSION]: v3.sliceMigration,
  [v3.VERSION]: v4.sliceMigration,
  [v4.VERSION]: v5.sliceMigration,
};

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});

export const anyStateZ = z
  .union([v5.stateZ, v4.stateZ, v3.stateZ, v2.stateZ, v1.stateZ, v0.stateZ])
  .transform((state) => migrateState(state));
