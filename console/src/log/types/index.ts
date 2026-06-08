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

import * as v0 from "@/log/types/v0";
import * as v1 from "@/log/types/v1";
import * as v2 from "@/log/types/v2";
import * as v3 from "@/log/types/v3";

export const stateZ = v3.stateZ;
export type State = v3.State;
export type SliceState = v3.SliceState;
export const ZERO_SLICE_STATE = v3.ZERO_SLICE_STATE;
export const ZERO_STATE = v3.ZERO_STATE;
export const ZERO_CHANNEL_ENTRY = v3.ZERO_CHANNEL_ENTRY;
export type ToolbarTab = v3.ToolbarTab;
export type ToolbarState = v3.ToolbarState;
export const ZERO_TOOLBAR_STATE = v3.ZERO_TOOLBAR_STATE;
export type PendingUpload = v3.PendingUpload;

export type AnyState = v0.State | v1.State | v2.State | v3.State;
export type AnySliceState =
  | v0.SliceState
  | v1.SliceState
  | v2.SliceState
  | v3.SliceState;

export const STATE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.stateMigration,
  [v1.VERSION]: v2.stateMigration,
  [v2.VERSION]: v3.stateMigration,
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
};

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});

export const anyStateZ = z
  .union([v3.stateZ, v2.stateZ, v1.stateZ, v0.stateZ])
  .transform((state) => migrateState(state));
