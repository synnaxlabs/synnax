// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";

import * as v0 from "@/user/types/v0";

export type SliceState = v0.SliceState;
export type AnySliceState = v0.SliceState;

export const ZERO_SLICE_STATE = v0.ZERO_SLICE_STATE;

const MIGRATIONS: migrate.Migrations = {};

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: "user.slice",
  migrations: MIGRATIONS,
  def: ZERO_SLICE_STATE,
});
