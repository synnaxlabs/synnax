// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v1 from "@/schematic/types/v1";
import * as v5 from "@/schematic/types/v5";

export const VERSION = "6.0.0";

export const legendStateZ = v1.legendStateZ
  .omit({ colors: true })
  .extend({ colors: z.record(z.string(), color.colorZ).default({}) });
export interface LegendState extends z.infer<typeof legendStateZ> {}
const ZERO_LEGEND_STATE: LegendState = {
  visible: true,
  position: { x: 50, y: 50, units: { x: "px", y: "px" } },
  colors: {},
};

export const stateZ = v5.stateZ
  .omit({ version: true, legend: true })
  .extend({ version: z.literal(VERSION), legend: legendStateZ });
export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = {
  ...v5.ZERO_STATE,
  version: VERSION,
  legend: ZERO_LEGEND_STATE,
};

export const sliceStateZ = v5.sliceStateZ
  .omit({ version: true, schematics: true })
  .extend({ version: z.literal(VERSION), schematics: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  ...v5.ZERO_SLICE_STATE,
  version: VERSION,
  schematics: {},
};

const migrateColors = (colors: Record<string, string>): Record<string, color.Color> =>
  Object.fromEntries(
    Object.entries(colors).map(([key, value]) => [key, color.construct(value)]),
  );

export const stateMigration = migrate.createMigration<v5.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...state,
    version: VERSION,
    legend: { ...state.legend, colors: migrateColors(state.legend.colors) },
  }),
});

export const sliceMigration = migrate.createMigration<v5.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ schematics, ...rest }) => ({
    ...rest,
    schematics: Object.fromEntries(
      Object.entries(schematics).map(([key, schematic]) => [
        key,
        stateMigration(schematic),
      ]),
    ),
    version: VERSION,
  }),
});
