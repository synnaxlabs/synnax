// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { Legend } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/schematic/migrations/v0";

export const VERSION = "1.0.0";
export type Version = typeof VERSION;

export const legendStateZ = z.object({
  visible: z.boolean(),
  position: Legend.stickyXYz,
});
export type LegendState = z.infer<typeof legendStateZ>;

const ZERO_LEGEND_STATE: LegendState = {
  visible: false,
  position: { x: 50, y: 50, units: { x: "px", y: "px" } },
};

export const stateZ = v0.stateZ
  .omit({ version: true })
  .extend({ version: z.literal(VERSION), legend: legendStateZ });

export interface State extends Omit<v0.State, "version"> {
  version: Version;
  legend: LegendState;
}

export const ZERO_STATE: State = {
  ...v0.ZERO_STATE,
  version: VERSION,
  legend: ZERO_LEGEND_STATE,
};

export const sliceStateZ = v0.sliceStateZ
  .omit({ version: true, schematics: true })
  .extend({ version: z.literal(VERSION), schematics: z.record(z.string(), stateZ) });

export interface SliceState extends Omit<v0.SliceState, "version" | "schematics"> {
  version: Version;
  schematics: Record<string, State>;
}

export const stateMigration = migrate.createMigration<v0.State, State>({
  name: "schematic.state",
  migrate: (input) => ({ ...input, legend: ZERO_LEGEND_STATE, version: VERSION }),
});

export const sliceMigration = migrate.createMigration<v0.SliceState, SliceState>({
  name: "schematic.slice",
  migrate: (input) => ({
    ...input,
    schematics: Object.fromEntries(
      Object.entries(input.schematics).map(([k, v]) => [k, stateMigration(v)]),
    ),
    version: VERSION,
  }),
});

export const ZERO_SLICE_STATE: SliceState = {
  ...v0.ZERO_SLICE_STATE,
  version: VERSION,
  schematics: {},
};
