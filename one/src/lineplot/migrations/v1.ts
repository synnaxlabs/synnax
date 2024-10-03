// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Legend } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/lineplot/migrations/v0";

export const legendStateZ = z.object({
  visible: z.boolean(),
  position: Legend.stickyXYz,
});

export type LegendState = z.infer<typeof legendStateZ>;

export const ZERO_LEGEND_STATE: LegendState = {
  visible: true,
  position: {
    x: 50,
    y: 50,
    root: { x: "left", y: "top" },
    units: { x: "px", y: "px" },
  },
};

export const stateZ = v0.stateZ
  .omit({
    legend: true,
    version: true,
  })
  .extend({
    version: z.literal("1.0.0"),
    legend: legendStateZ,
  });

export type State = z.infer<typeof stateZ>;

export const ZERO_STATE: State = {
  ...v0.ZERO_STATE,
  version: "1.0.0",
  legend: ZERO_LEGEND_STATE,
};

export const sliceStateZ = v0.sliceStateZ
  .omit({
    plots: true,
    version: true,
  })
  .extend({
    version: z.literal("1.0.0"),
    plots: z.record(stateZ),
  });

export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  ...v0.ZERO_SLICE_STATE,
  version: "1.0.0",
  plots: {},
};

export const stateMigration = migrate.createMigration<v0.State, State>({
  name: "lineplot.state",
  migrate: (s) => ({ ...s, version: "1.0.0", legend: ZERO_LEGEND_STATE }),
});

export const sliceMigration = migrate.createMigration<v0.SliceState, SliceState>({
  name: "lineplot.slice",
  migrate: (s) => ({
    ...s,
    version: "1.0.0",
    plots: Object.fromEntries(
      Object.keys(s.plots).map((k) => [k, stateMigration(s.plots[k])]),
    ),
  }),
});
