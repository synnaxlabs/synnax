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

import { type SliceState as LayoutSliceState } from "@/session/layout/slice";
import * as v0 from "@/project/services/layoutMigrations/v0";
import * as v1 from "@/project/services/layoutMigrations/v1";
import * as v2 from "@/project/services/layoutMigrations/v2";
import * as v3 from "@/project/services/layoutMigrations/v3";
import * as v4 from "@/project/services/layoutMigrations/v4";
import * as v5 from "@/project/services/layoutMigrations/v5";
import * as v6 from "@/project/services/layoutMigrations/v6";
import * as v7 from "@/project/services/layoutMigrations/v7";
import * as v8 from "@/project/services/layoutMigrations/v8";
import * as v9 from "@/project/services/layoutMigrations/v9";
import * as v10 from "@/project/services/layoutMigrations/v10";
import * as v11 from "@/project/services/layoutMigrations/v11";

export type State<A = unknown> = v0.State<A>;
export type SliceState = v11.SliceState;
export type WindowProps = v0.WindowProps;
export type AnyState<A = unknown> = v0.State<A>;
export type AnySliceState =
  | v0.SliceState
  | v1.SliceState
  | v2.SliceState
  | v3.SliceState
  | v4.SliceState
  | v5.SliceState
  | v6.SliceState
  | v7.SliceState
  | v8.SliceState
  | v9.SliceState
  | v10.SliceState
  | v11.SliceState;

export const SLICE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.sliceMigration,
  [v1.VERSION]: v2.sliceMigration,
  [v2.VERSION]: v3.sliceMigration,
  [v3.VERSION]: v4.sliceMigration,
  [v4.VERSION]: v5.sliceMigration,
  [v5.VERSION]: v6.sliceMigration,
  [v6.VERSION]: v7.sliceMigration,
  [v7.VERSION]: v8.sliceMigration,
  [v8.VERSION]: v9.sliceMigration,
  [v9.VERSION]: v10.sliceMigration,
  [v10.VERSION]: v11.sliceMigration,
};

export const ZERO_SLICE_STATE = v11.ZERO_SLICE_STATE;
export const ZERO_MOSAIC_STATE = v0.ZERO_MOSAIC_STATE;
export const MAIN_LAYOUT = v0.MAIN_LAYOUT;

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});

export const anySliceStateZ = z
  .union([
    v11.sliceStateZ,
    v10.sliceStateZ,
    v9.sliceStateZ,
    v8.sliceStateZ,
    v7.sliceStateZ,
    v6.sliceStateZ,
    v5.sliceStateZ,
    v4.sliceStateZ,
    v3.sliceStateZ,
    v2.sliceStateZ,
    v1.sliceStateZ,
    v0.sliceStateZ,
  ])
  .transform((state) => migrateSlice(state));

/**
 * Migrates a persisted or exported layout slice of any historical version up to the
 * current layout slice state, used when importing a project from disk or loading a
 * project saved against an older schema.
 */
export const migrateLayout = (data: unknown): LayoutSliceState => {
  const { version: _version, ...rest } = anySliceStateZ.parse(data);
  return { ...rest, version: 0 };
};
