// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";

import * as v0 from "@/range/migrations/v0";

export type SliceState = v0.SliceState;
export type Range = v0.Range;
export type DynamicRange = v0.DynamicRange;
export type StaticRange = v0.StaticRange;
export const ZERO_SLICE_STATE = v0.ZERO_SLICE_STATE;

export const MIGRATIONS: migrate.Migrations = {};
export const migrateSlice = migrate.migrator<SliceState, SliceState>({
  name: "range.slice",
  migrations: MIGRATIONS,
  def: ZERO_SLICE_STATE,
});
