// Copyright 2025 Synnax Labs, Inc.
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

export const titleStateZ = v0.titleStateZ;
export type TitleState = v0.TitleState;
export const ZERO_TITLE_STATE = v0.ZERO_TITLE_STATE;

export const legendStateZ = v1.legendStateZ;
export type LegendState = v1.LegendState;
export const ZERO_LEGEND_STATE = v1.ZERO_LEGEND_STATE;

export const viewportStateZ = v0.viewportStateZ;
export type ViewportState = v0.ViewportState;
export const ZERO_VIEWPORT_STATE = v0.ZERO_VIEWPORT_STATE;

export const selectionStateZ = v0.selectionStateZ;
export type SelectionState = v0.SelectionState;
export const ZERO_SELECTION_STATE = v0.ZERO_SELECTION_STATE;

export type AxisState = v2.AxisState;
export const ZERO_AXIS_STATE = v2.ZERO_AXIS_STATE;

export type AxesState = v2.AxesState;
export const ZERO_AXES_STATE = v2.ZERO_AXES_STATE;

export const lineStateZ = v0.lineStateZ;
export type LineState = v0.LineState;
export const ZERO_LINE_STATE = v0.ZERO_LINE_STATE;

export const linesStateZ = v0.linesStateZ;
export type LinesState = v0.LinesState;
export const ZERO_LINES_STATE = v0.ZERO_LINES_STATE;

export const ruleStateZ = v0.ruleStateZ;
export type RuleState = v0.RuleState;
export const ZERO_RULE_STATE = v0.ZERO_RULE_STATE;

export const rulesStateZ = v0.rulesStateZ;
export type RulesState = v0.RulesState;
export const ZERO_RULES_STATE = v0.ZERO_RULES_STATE;

export const channelsStateZ = v0.channelsStateZ;
export type ChannelsState = v0.ChannelsState;
export const ZERO_CHANNELS_STATE = v0.ZERO_CHANNELS_STATE;

export const rangesStateZ = v0.rangesStateZ;
export type RangesState = v0.RangesState;
export const ZERO_RANGES_STATE = v0.ZERO_RANGES_STATE;

export const stateZ = v4.stateZ;
export type State = v4.State;
export const ZERO_STATE = v4.ZERO_STATE;

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

export const sliceStateZ = v4.sliceStateZ;
export type SliceState = v4.SliceState;
export const ZERO_SLICE_STATE = v4.ZERO_SLICE_STATE;

export type AnyState = v0.State | v1.State | v2.State | v3.State | v4.State;
export type AnySliceState =
  | v0.SliceState
  | v1.SliceState
  | v2.SliceState
  | v3.SliceState
  | v4.SliceState;

export const STATE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.stateMigration,
  [v1.VERSION]: v2.stateMigration,
  [v2.VERSION]: v3.stateMigration,
  [v3.VERSION]: v4.stateMigration,
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
};

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});

export const anyStateZ = z
  .union([v4.stateZ, v3.stateZ, v2.stateZ, v1.stateZ, v0.stateZ])
  .transform((state) => migrateState(state));
