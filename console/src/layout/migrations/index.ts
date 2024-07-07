import { migrate } from "@synnaxlabs/x";

import * as v0 from "@/layout/migrations/v0";
import * as v3 from "@/layout/migrations/v3";

export type State<A = any> = v0.State<A>;
export type SliceState = v3.SliceState;
export type NavDrawerLocation = v0.NavDrawerLocation;
export type NavDrawerEntryState = v0.NavDrawerEntryState;
export type WindowProps = v0.WindowProps;

export const SLICE_MIGRATIONS: migrate.Migrations = {
  "0.0.0": v3.sliceMigration,
  "0.1.0": v3.sliceMigration,
  "0.2.0": v3.sliceMigration,
};

export const ZERO_SLICE_STATE = v3.ZERO_SLICE_STATE;
export const ZERO_MOSAIC_STATE = v0.ZERO_MOSAIC_STATE;
export const MAIN_LAYOUT = v0.MAIN_LAYOUT;

export const migrateSlice = migrate.migrator({
  name: "layout.slice",
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});
