// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";

import * as v0 from "@/version/types/v0";
import * as v1 from "@/version/types/v1";

export type SliceState = v1.SliceState;
export type AnySliceState = v0.SliceState | v1.SliceState;
export const ZERO_SLICE_STATE = v1.ZERO_SLICE_STATE;

export const SLICE_MIGRATIONS: migrate.Migrations = {};

// Because the v0 state had a key called version, the usual migration pattern from X
// does not work. Instead, we need to check if the state is a v0 state, manually convert
// it to a v1 state, and then run the internal migrator.

const internalMigrator = migrate.migrator<AnySliceState, SliceState>({
  name: "version.slice",
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});

const v0StrictSliceStateZ = v0.sliceStateZ.strict();

export const migrateSlice: (v: AnySliceState) => SliceState = (v) => {
  const state = v0StrictSliceStateZ.safeParse(v).success ? v1.migrate(v) : v;
  return internalMigrator(state);
};
