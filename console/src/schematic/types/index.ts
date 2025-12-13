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

import * as v0 from "@/schematic/types/v0";
import * as v1 from "@/schematic/types/v1";
import * as v2 from "@/schematic/types/v2";
import * as v3 from "@/schematic/types/v3";
import * as v4 from "@/schematic/types/v4";
import * as v5 from "@/schematic/types/v5";
import * as v6 from "@/schematic/types/v6";

export type NodeProps = v0.NodeProps;
export type EdgeProps = v0.EdgeProps;
export type State = v6.State;
export type SliceState = v6.SliceState;
export type ToolbarTab = v0.ToolbarTab;
export type ToolbarState = v0.ToolbarState;
export type LegendState = v1.LegendState;
export type CopyBuffer = v0.CopyBuffer;
export type AnyState =
  | v0.State
  | v1.State
  | v2.State
  | v3.State
  | v4.State
  | v5.State
  | v6.State;
export type AnySliceState =
  | v0.SliceState
  | v1.SliceState
  | v2.SliceState
  | v3.SliceState
  | v4.SliceState
  | v5.SliceState
  | v6.SliceState;

export const ZERO_STATE = v6.ZERO_STATE;
export const ZERO_SLICE_STATE = v6.ZERO_SLICE_STATE;

const STATE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.stateMigration,
  [v1.VERSION]: v2.stateMigration,
  [v2.VERSION]: v3.stateMigration,
  [v3.VERSION]: v4.stateMigration,
  [v4.VERSION]: v5.stateMigration,
  [v5.VERSION]: v6.stateMigration,
};

const SLICE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.sliceMigration,
  [v1.VERSION]: v2.sliceMigration,
  [v2.VERSION]: v3.sliceMigration,
  [v3.VERSION]: v4.sliceMigration,
  [v4.VERSION]: v5.sliceMigration,
  [v5.VERSION]: v6.sliceMigration,
};

export const migrateState = migrate.migrator<AnyState, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrations: STATE_MIGRATIONS,
  def: ZERO_STATE,
});

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});

export const anyStateZ = z
  .union([v6.stateZ, v5.stateZ, v4.stateZ, v3.stateZ, v2.stateZ, v1.stateZ, v0.stateZ])
  .transform((state) => migrateState(state));
