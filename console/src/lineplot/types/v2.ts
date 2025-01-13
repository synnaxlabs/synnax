// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { axis } from "@synnaxlabs/pluto";
import { bounds, migrate } from "@synnaxlabs/x";
import { z } from "zod";

import { X_AXIS_KEYS, type XAxisKey } from "@/lineplot/axis";
import * as v0 from "@/lineplot/types/v0";
import * as v1 from "@/lineplot/types/v1";

// V2 IS DEFINED FOR SYNNAX V0.25

export const axisStateZ = v0.axisStateZ.extend({
  type: axis.tickType.optional(),
});

export type AxisState = z.infer<typeof axisStateZ>;

export const axesStateZ = v0.axesStateZ.omit({ axes: true }).extend({
  axes: z.object({
    y1: axisStateZ,
    y2: axisStateZ,
    y3: axisStateZ,
    y4: axisStateZ,
    x1: axisStateZ,
    x2: axisStateZ,
  }),
});

export type AxesState = z.infer<typeof axesStateZ>;

export const ZERO_AXIS_STATE: AxisState = {
  key: "x1",
  label: "",
  labelDirection: "x",
  labelLevel: "small",
  bounds: bounds.ZERO,
  autoBounds: { lower: true, upper: true },
  tickSpacing: 75,
};

export const ZERO_AXES_STATE: AxesState = {
  renderTrigger: 0,
  hasHadChannelSet: false,
  axes: {
    y1: { ...ZERO_AXIS_STATE, key: "y1", labelDirection: "y" },
    y2: { ...ZERO_AXIS_STATE, key: "y2", labelDirection: "y" },
    y3: { ...ZERO_AXIS_STATE, key: "y3", labelDirection: "y" },
    y4: { ...ZERO_AXIS_STATE, key: "y4", labelDirection: "y" },
    x1: { ...ZERO_AXIS_STATE, key: "x1", type: "time" },
    x2: { ...ZERO_AXIS_STATE, key: "x2", type: "time" },
  },
};

export const stateZ = v1.stateZ
  .omit({
    axes: true,
    version: true,
  })
  .extend({
    version: z.literal("2.0.0"),
    axes: axesStateZ,
  });

export type State = z.infer<typeof stateZ>;

export const ZERO_STATE: State = {
  ...v1.ZERO_STATE,
  version: "2.0.0",
  axes: ZERO_AXES_STATE,
};

export const sliceStateZ = v1.sliceStateZ
  .omit({
    plots: true,
    version: true,
  })
  .extend({
    version: z.literal("2.0.0"),
    plots: z.record(stateZ),
  });

export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  ...v1.ZERO_SLICE_STATE,
  version: "2.0.0",
  plots: {},
};

export const stateMigration = migrate.createMigration<v1.State, State>({
  name: "lineplot.state",
  migrate: (s) => ({
    ...s,
    version: "2.0.0",
    axes: {
      ...s.axes,
      axes: Object.fromEntries(
        Object.entries(s.axes.axes).map(([key, value]) => {
          if (!X_AXIS_KEYS.includes(key as XAxisKey))
            return [key, { ...value, labelDirection: "y" }];
          return [key, { ...value, type: "time" }];
        }),
      ) as AxesState["axes"],
    },
  }),
});

export const sliceMigration = migrate.createMigration<v1.SliceState, SliceState>({
  name: "lineplot.slice",
  migrate: (s) => ({
    ...s,
    version: "2.0.0",
    plots: Object.fromEntries(
      Object.entries(s.plots).map(([key, value]) => [key, stateMigration(value)]),
    ),
  }),
});
