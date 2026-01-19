// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { axis } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/lineplot/types/v0";
import * as v1 from "@/lineplot/types/v1";

export const VERSION = "2.0.0";

export const axisStateZ = v0.axisStateZ.extend({ type: axis.tickType.optional() });
export interface AxisState extends z.infer<typeof axisStateZ> {}
export const ZERO_AXIS_STATE: AxisState = { ...v0.ZERO_AXIS_STATE };

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
export interface AxesState extends z.infer<typeof axesStateZ> {}
export const ZERO_AXES_STATE: AxesState = {
  ...v0.ZERO_AXES_STATE,
  axes: {
    y1: { ...v0.ZERO_AXES_STATE.axes.y1, labelDirection: "y" },
    y2: { ...v0.ZERO_AXES_STATE.axes.y2, labelDirection: "y" },
    y3: { ...v0.ZERO_AXES_STATE.axes.y3, labelDirection: "y" },
    y4: { ...v0.ZERO_AXES_STATE.axes.y4, labelDirection: "y" },
    x1: { ...v0.ZERO_AXES_STATE.axes.x1, type: "time" },
    x2: { ...v0.ZERO_AXES_STATE.axes.x2, type: "time" },
  },
};

export const stateZ = v1.stateZ
  .omit({ axes: true, version: true })
  .extend({ version: z.literal(VERSION), axes: axesStateZ });
export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = {
  ...v1.ZERO_STATE,
  version: VERSION,
  axes: ZERO_AXES_STATE,
};

export const sliceStateZ = v1.sliceStateZ
  .omit({ plots: true, version: true })
  .extend({ version: z.literal(VERSION), plots: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  ...v1.ZERO_SLICE_STATE,
  version: VERSION,
  plots: {},
};

export const stateMigration = migrate.createMigration<v1.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: ({ axes, ...rest }) => ({
    ...rest,
    version: VERSION,
    axes: {
      ...axes,
      axes: Object.fromEntries(
        Object.entries(axes.axes).map(([key, axis]) => {
          if (key.startsWith("x")) return [key, { ...axis, type: "time" }];
          return [key, { ...axis, labelDirection: "y" }];
        }),
      ),
    },
  }),
});

export const sliceMigration = migrate.createMigration<v1.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ plots, ...rest }) => ({
    ...rest,
    version: VERSION,
    plots: Object.fromEntries(
      Object.entries(plots).map(([key, plot]) => [key, stateMigration(plot)]),
    ),
  }),
});
