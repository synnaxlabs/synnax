// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";

import * as v0 from "@/workspace/types/v0";
import * as v1 from "@/workspace/types/v1";

export type Workspace = v1.Workspace;
export type SliceState = v1.SliceState;
export type AnySliceState = v0.SliceState | v1.SliceState;

export const ZERO_SLICE_STATE = v1.ZERO_SLICE_STATE;

const SLICE_MIGRATIONS: migrate.Migrations = { [v0.VERSION]: v1.sliceMigration };

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});
