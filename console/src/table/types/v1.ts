// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { migrate, type record } from "@synnaxlabs/x";
import { z } from "zod";

import type * as v0 from "@/table/types/v0";

export const VERSION = "1.0.0";

// pendingUploadZ is the table payload needed to upload a legacy v0 table on
// first render. name is omitted because the live name lives in Layout.
const pendingUploadZ = table.tableZ.omit({ name: true });
interface PendingUpload extends z.infer<typeof pendingUploadZ> {}

// v1 removes the structural model (layout / cells) from console state; rows,
// columns, and cells live in the Pluto-owned flux store, and the slice only
// carries UI state. Selection moves from a per-cell `selected: boolean` to a
// per-table `selectedCells: string[]` array. Legacy v0 input has its
// structural data parked in pendingUpload so consumers can re-sync it to the
// server on first render.
export const stateZ = z.object({
  key: z.string(),
  version: z.literal(VERSION),
  lastSelected: z.string().nullable(),
  editable: z.boolean(),
  selectedCells: z.array(z.string()).default([]),
  // hideIndicators controls whether the row/column indicator strips are
  // hidden when editable is false. Setting only takes effect outside edit
  // mode; entering edit mode always shows them.
  hideIndicators: z.boolean().default(false),
  pendingUpload: pendingUploadZ.optional(),
});

export interface State extends z.infer<typeof stateZ> {}

export const ZERO_STATE: State = {
  key: "",
  version: VERSION,
  lastSelected: null,
  editable: true,
  selectedCells: [],
  hideIndicators: false,
  pendingUpload: undefined,
};

export const sliceStateZ = z.object({
  version: z.literal(VERSION),
  tables: z.record(z.string(), stateZ),
});

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  tables: {},
};

export const STATE_MIGRATION_NAME = "table.state";
export const SLICE_MIGRATION_NAME = "table.slice";

const buildPendingUpload = (state: v0.State): PendingUpload => ({
  key: state.key,
  rows: state.layout.rows.map((r) => ({
    size: r.size,
    cells: r.cells.map((c) => c.key),
  })),
  columns: state.layout.columns,
  cells: Object.fromEntries(
    Object.entries(state.cells).map(([k, c]) => [
      k,
      {
        key: c.key,
        variant: c.variant,
        props: (c.props as record.Unknown) ?? {},
      },
    ]),
  ),
});

// Drops the per-cell selected flag and parks v0's structural model in
// pendingUpload when the table is not yet remoteCreated. remoteCreated tables
// already have authoritative data on the server, so pendingUpload stays
// undefined for those.
export const stateMigration = migrate.createMigration<v0.State, State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    key: state.key,
    version: VERSION,
    lastSelected: state.lastSelected,
    editable: state.editable,
    selectedCells: [],
    hideIndicators: false,
    pendingUpload: state.remoteCreated ? undefined : buildPendingUpload(state),
  }),
});

export const sliceMigration = migrate.createMigration<v0.SliceState, SliceState>({
  name: SLICE_MIGRATION_NAME,
  migrate: ({ tables }) => ({
    version: VERSION,
    tables: Object.fromEntries(
      Object.entries(tables).map(([k, t]) => [k, stateMigration(t)]),
    ),
  }),
});
