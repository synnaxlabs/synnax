// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate, sticky } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/lineplot/types/v0";

export const VERSION = "1.0.0";

export const legendStateZ = v0.legendStateZ.extend({ position: sticky.xy });
export interface LegendState extends z.infer<typeof legendStateZ> {}
export const ZERO_LEGEND_STATE: LegendState = {
  ...v0.ZERO_LEGEND_STATE,
  position: {
    x: 50,
    y: 50,
    root: { x: "left", y: "top" },
    units: { x: "px", y: "px" },
  },
};

export const stateZ = v0.stateZ
  .omit({ legend: true, version: true })
  .extend({ version: z.literal(VERSION), legend: legendStateZ });
export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = {
  ...v0.ZERO_STATE,
  version: VERSION,
  legend: ZERO_LEGEND_STATE,
};

export const sliceStateZ = v0.sliceStateZ
  .omit({ plots: true, version: true })
  .extend({ version: z.literal(VERSION), plots: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  ...v0.ZERO_SLICE_STATE,
  version: VERSION,
  plots: {},
};

export const STATE_MIGRATION_NAME = "lineplot.state";
export const SLICE_MIGRATION_NAME = "lineplot.slice";

export const stateMigration = migrate.createMigration<v0.State, State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({ ...state, version: VERSION, legend: ZERO_LEGEND_STATE }),
});

export const sliceMigration = migrate.createMigration<v0.SliceState, SliceState>({
  name: SLICE_MIGRATION_NAME,
  migrate: ({ plots, ...rest }) => ({
    ...rest,
    version: VERSION,
    plots: Object.fromEntries(
      Object.entries(plots).map(([key, plot]) => [key, stateMigration(plot)]),
    ),
  }),
});
