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
import * as v4 from "@/lineplot/types/v4";

export const VERSION = "5.0.0";

// pendingUploadZ stages a plot's body on the client until it has landed on
// the server. Each field is independently optional so callers can stage
// subsets.
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

export const stateZ = v4.stateZ
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
    selectedRules: z.array(z.string()).default([]),
    pendingUpload: pendingUploadZ.optional(),
  });

export interface State extends z.infer<typeof stateZ> {}

// Destructure rather than relying on the type-level Omit; a plain spread
// would carry the v4 body fields at runtime and break deep equality with the
// migrated shape.
const {
  title: _title,
  legend: _legend,
  channels: _channels,
  ranges: _ranges,
  axes: _axes,
  lines: _lines,
  rules: _rules,
  version: _version,
  ...zeroRest
} = v4.ZERO_STATE;

export const ZERO_STATE: State = {
  ...zeroRest,
  version: VERSION,
  selectedRules: [],
  pendingUpload: undefined,
};

export const sliceStateZ = v4.sliceStateZ
  .omit({ plots: true, version: true })
  .extend({ version: z.literal(VERSION), plots: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = { version: VERSION, plots: {} };

// buildPendingUpload projects v4's hand-rolled body fields into the
// oracle-typed PendingUpload shape. Drops the per-rule `selected` flag
// since selection lifts to selectedRules at the slice level.
const buildPendingUpload = (state: v4.State): PendingUpload => ({
  title: lineplot.titleZ.parse(state.title),
  legend: lineplot.legendZ.parse(state.legend),
  channels: lineplot.channelsZ.parse(state.channels),
  ranges: lineplot.rangesZ.parse(state.ranges),
  axes: lineplot.axesZ.parse(state.axes.axes),
  lines: state.lines.map((l) => lineplot.lineZ.parse(l)),
  rules: state.rules.map((r) => {
    const { selected: _selected, ...rest } = r;
    return lineplot.ruleZ.parse(rest);
  }),
});

// stateMigration drops the v4 body fields. remoteCreated plots are
// authoritative on the server already, so pendingUpload stays undefined.
// Non-remoteCreated plots are projected into pendingUpload. Selection on
// rules lifts from a per-rule `selected` flag to a per-plot selectedRules
// array.
export const stateMigration = migrate.createMigration<v4.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => {
    const selectedRules = state.rules
      .filter((r) => r.selected === true)
      .map((r) => r.key);
    const {
      title: _title,
      legend: _legend,
      channels: _channels,
      ranges: _ranges,
      axes: _axes,
      lines: _lines,
      rules: _rules,
      ...rest
    } = state;
    return {
      ...rest,
      version: VERSION,
      selectedRules,
      pendingUpload: state.remoteCreated ? undefined : buildPendingUpload(state),
    };
  },
});

export const sliceMigration = migrate.createMigration<v4.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ plots }) => ({
    version: VERSION,
    plots: Object.fromEntries(
      Object.entries(plots).map(([key, plot]) => [key, stateMigration(plot)]),
    ),
  }),
});
