// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate, sticky } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/schematic/types/v0";

export const VERSION = "1.0.0";

export const legendStateZ = z.object({
  visible: z.boolean(),
  position: sticky.xy,
});
export interface LegendState extends z.infer<typeof legendStateZ> {}
const ZERO_LEGEND_STATE: LegendState = {
  visible: true,
  position: { x: 50, y: 50, units: { x: "px", y: "px" } },
};

export const stateZ = v0.stateZ
  .omit({ version: true })
  .extend({ version: z.literal(VERSION), legend: legendStateZ });

export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = {
  ...v0.ZERO_STATE,
  version: VERSION,
  legend: ZERO_LEGEND_STATE,
};

export const sliceStateZ = v0.sliceStateZ
  .omit({ version: true, schematics: true })
  .extend({ version: z.literal(VERSION), schematics: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  ...v0.ZERO_SLICE_STATE,
  version: VERSION,
  schematics: {},
};

export const STATE_MIGRATION_NAME = "schematic.state";
export const SLICE_MIGRATION_NAME = "schematic.slice";

export const stateMigration = migrate.createMigration<v0.State, State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({ ...state, legend: ZERO_LEGEND_STATE, version: VERSION }),
});

export const sliceMigration = migrate.createMigration<v0.SliceState, SliceState>({
  name: SLICE_MIGRATION_NAME,
  migrate: ({ schematics, ...rest }) => ({
    ...rest,
    version: VERSION,
    schematics: Object.fromEntries(
      Object.entries(schematics).map(([key, schematic]) => [
        key,
        stateMigration(schematic),
      ]),
    ),
  }),
});
