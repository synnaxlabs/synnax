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

// Destructure rather than relying on the type-level Omit; a plain spread
// would carry the body fields at runtime and break deep equality with the
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
} = v5.ZERO_STATE;

export const ZERO_STATE: State = {
  ...zeroRest,
  version: VERSION,
  pendingUpload: undefined,
};

export const sliceStateZ = v5.sliceStateZ
  .omit({ plots: true, version: true })
  .extend({ version: z.literal(VERSION), plots: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = { version: VERSION, plots: {} };

// stateMigration drops the body fields entirely. Users upgrading from v5
// have the canonical body on the server already (it was synced there via
// SetData in the v5 flow), so re-uploading via pendingUpload would 409 on
// duplicate key. Importers that need the body to land on the server
// construct pendingUpload explicitly themselves (see services/import.ts).
export const stateMigration = migrate.createMigration<v5.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => {
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
    return { ...rest, version: VERSION, pendingUpload: undefined };
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
