// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";
import { z } from "zod/v4";

import * as v0 from "@/arc/types/v0";
import * as v1 from "@/arc/types/v1";
import * as v2 from "@/arc/types/v2";

export type NodeProps = v2.NodeProps;
export type State = v2.State;
export type SliceState = v2.SliceState;
export type ToolbarTab = v2.ToolbarTab;
export type ToolbarState = v2.ToolbarState;
export type GraphState = v2.GraphState;
export type CopyBuffer = v2.CopyBuffer;
export type AnyState = v0.State | v1.State | v2.State;
export type AnySliceState = v0.SliceState | v1.SliceState | v2.SliceState;
export type Mode = v2.Mode;

export const TYPE = v2.TYPE;
export const ZERO_STATE = v2.ZERO_STATE;
export const ZERO_SLICE_STATE = v2.ZERO_SLICE_STATE;

const STATE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.stateMigration,
  [v1.VERSION]: v2.stateMigration,
};

const SLICE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.sliceMigration,
  [v1.VERSION]: v2.sliceMigration,
};

export const migrateState = migrate.migrator<AnyState, State>({
  name: v2.STATE_MIGRATION_NAME,
  migrations: STATE_MIGRATIONS,
  def: ZERO_STATE,
});

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: v2.SLICE_MIGRATION_NAME,
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});

export const anyStateZ = z
  .union([v2.stateZ, v1.stateZ, v0.stateZ])
  .transform((state) => migrateState(state));
