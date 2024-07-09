// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";

import * as v0 from "@/schematic/migrations/v0";
import * as v1 from "@/schematic/migrations/v1";

export type NodeProps = v0.NodeProps;
export type State = v1.State;
export type SliceState = v1.SliceState;
export type ToolbarTab = v0.ToolbarTab;
export type ToolbarState = v0.ToolbarState;
export type LegendState = v1.LegendState;
export type CopyBuffer = v0.CopyBuffer;
export type AnyState = v0.State | v1.State;
export type AnySliceState = v0.SliceState | v1.SliceState;

export const ZERO_STATE = v1.ZERO_STATE;
export const ZERO_SLICE_STATE = v1.ZERO_SLICE_STATE;

export const STATE_MIGRATIONS: migrate.Migrations = {
  "0.0.0": v1.stateMigration,
};

export const SLICE_MIGRATIONS: migrate.Migrations = {
  "0.0.0": v1.sliceMigration,
};

export const migrateState = migrate.migrator<v1.State>({
  name: "schematic.state",
  migrations: STATE_MIGRATIONS,
  def: v1.ZERO_STATE,
});

export const migrateSlice = migrate.migrator<v1.SliceState>({
  name: "schematic.slice",
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});
