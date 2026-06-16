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

import * as v0 from "@/table/types/v0";
import * as v1 from "@/table/types/v1";

export const stateZ = v1.stateZ;
export type State = v1.State;
export type SliceState = v1.SliceState;
export const ZERO_STATE = v1.ZERO_STATE;
export const ZERO_SLICE_STATE = v1.ZERO_SLICE_STATE;

export type AnyState = v0.State | v1.State;
export type AnySliceState = v0.SliceState | v1.SliceState;

export const STATE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.stateMigration,
};

export const migrateState = migrate.migrator<AnyState, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrations: STATE_MIGRATIONS,
  def: ZERO_STATE,
});

export const SLICE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.sliceMigration,
};

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});

// anyStateZ parses either a v0 or v1 state blob and returns a uniform v1
// State. v0 inputs flow through the migration ladder, so pendingUpload is
// populated when the source carried unsynced structural data.
export const anyStateZ = z.union([v1.stateZ, v0.stateZ]).transform(migrateState);
