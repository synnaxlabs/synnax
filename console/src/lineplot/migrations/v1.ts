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

export const stateMigration: migrate.Migration<typeof v0.stateZ, typeof stateZ> = {
  input: v0.stateZ,
  output: stateZ,
  migrate: (s) => ({ ...s, version: "1.0.0", legend: ZERO_LEGEND_STATE }),
};

export const sliceMigration: migrate.Migration<
  typeof v0.sliceStateZ,
  typeof sliceStateZ
> = {
  input: v0.sliceStateZ,
  output: sliceStateZ,
  migrate: (s) => {
    console.log("Migrating slice state from", s.version, s.plots);
    return {
      ...s,
      version: "1.0.0",
      plots: Object.fromEntries(
        Object.keys(s.plots).map((k) => [k, stateMigration.migrate(s.plots[k])]),
      ),
    };
  },
};
