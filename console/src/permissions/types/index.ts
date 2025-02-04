// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";

import * as v0 from "@/permissions/types/v0";

export type SliceState = v0.State;
export type AnySliceState = v0.State;
export const ALLOW_ALL = v0.ALLOW_ALL;

export const ZERO_SLICE_STATE = v0.ZERO_STATE;

const MIGRATIONS: migrate.Migrations = {};

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: "permissions.slice",
  migrations: MIGRATIONS,
  def: ZERO_SLICE_STATE,
});
