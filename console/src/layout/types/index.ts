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

import * as v0 from "@/layout/types/v0";
import * as v1 from "@/layout/types/v1";
import * as v2 from "@/layout/types/v2";
import * as v3 from "@/layout/types/v3";
import * as v4 from "@/layout/types/v4";
import * as v5 from "@/layout/types/v5";
import * as v6 from "@/layout/types/v6";
import * as v7 from "@/layout/types/v7";

export type State<A = unknown> = v0.State<A>;
export type SliceState = v7.SliceState;
export type NavDrawerLocation = v0.NavDrawerLocation;
export type NavDrawerEntryState = v0.NavDrawerEntryState;
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
  | v7.SliceState;

export const SLICE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.sliceMigration,
  [v1.VERSION]: v2.sliceMigration,
  [v2.VERSION]: v3.sliceMigration,
  [v3.VERSION]: v4.sliceMigration,
  [v4.VERSION]: v5.sliceMigration,
  [v5.VERSION]: v6.sliceMigration,
  [v6.VERSION]: v7.sliceMigration,
};

export const ZERO_SLICE_STATE = v7.ZERO_SLICE_STATE;
export const ZERO_MOSAIC_STATE = v0.ZERO_MOSAIC_STATE;
export const MAIN_LAYOUT = v0.MAIN_LAYOUT;

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});

export const anySliceStateZ = z
  .union([
    v6.sliceStateZ,
    v7.sliceStateZ,
    v5.sliceStateZ,
    v4.sliceStateZ,
    v3.sliceStateZ,
    v2.sliceStateZ,
    v1.sliceStateZ,
    v0.sliceStateZ,
  ])
  .transform((state) => migrateSlice(state));
