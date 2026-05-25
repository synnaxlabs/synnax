// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v1 from "@/lineplot/types/v1";
import * as v5 from "@/lineplot/types/v5";

export const VERSION = "6.0.0";

// pendingUploadZ carries the body of a plot that was created locally but has
// not yet been uploaded to the server. After a successful upload it is
// cleared and Pluto's flux store becomes the canonical source for the body.
// Each body field is independently optional so callers can stage only the
// pieces they care about (e.g., a new plot opened from a single range only
// sets `ranges`).
const pendingUploadZ = z.object({
  title: lineplot.titleZ.optional(),
  legend: lineplot.legendZ.optional(),
  channels: lineplot.channelsZ.optional(),
  ranges: lineplot.rangesZ.optional(),
  axes: lineplot.axesZ.optional(),
  lines: z.array(lineplot.lineZ).optional(),
  rules: z.array(lineplot.ruleZ).optional(),
});

export type PendingUpload = z.infer<typeof pendingUploadZ>;

export const stateZ = v5.stateZ
  .omit({
    version: true,
    title: true,
    legend: true,
    channels: true,
    ranges: true,
    axes: true,
    lines: true,
    rules: true,
  })
  .extend({
    version: z.literal(VERSION),
    pendingUpload: pendingUploadZ.optional(),
  });

export interface State extends z.infer<typeof stateZ> {}

export const ZERO_STATE: State = {
  ...(v5.ZERO_STATE as Omit<
    v5.State,
    "title" | "legend" | "channels" | "ranges" | "axes" | "lines" | "rules" | "version"
  >),
  version: VERSION,
  pendingUpload: undefined,
};

export const sliceStateZ = v5.sliceStateZ
  .omit({ plots: true, version: true })
  .extend({ version: z.literal(VERSION), plots: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = { version: VERSION, plots: {} };

export const stateMigration = migrate.createMigration<v5.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => {
    const {
      title,
      legend,
      channels,
      ranges,
      axes: { axes: axesMap },
      lines,
      rules,
      ...rest
    } = state;
    return {
      ...rest,
      version: VERSION,
      pendingUpload: {
        title,
        legend,
        channels,
        ranges,
        axes: axesMap,
        lines,
        rules,
      },
    };
  },
});

export const sliceMigration = migrate.createMigration<v5.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ plots }) => ({
    version: VERSION,
    plots: Object.fromEntries(
      Object.entries(plots).map(([key, plot]) => [key, stateMigration(plot)]),
    ),
  }),
});
