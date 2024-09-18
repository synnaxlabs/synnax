import { migrate } from "@synnaxlabs/x";

import * as v0 from "@/policies/migrations/v0";

export type SliceState = v0.State;
export type AnySliceState = v0.State;

export const ZERO_SLICE_STATE = v0.ZERO_STATE;

const MIGRATIONS: migrate.Migrations = {};

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: "policies.slice",
  migrations: MIGRATIONS,
  def: ZERO_SLICE_STATE,
});
