// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import type * as v0 from "@/table/types/v0";
import { toWire } from "@/table/types/v0";

export const VERSION = "1.0.0";

// pendingUploadZ carries the structural data a v0 table needs uploaded to
// flux/server on first render. name is supplied at upload time from Layout.
export const pendingUploadZ = table.tableZ.omit({ name: true });
export interface PendingUpload extends z.infer<typeof pendingUploadZ> {}

// v1 removes the structural model (layout / cells) from console state. Rows,
// columns, and cells now live in the Pluto-owned flux store keyed by table
// key; the slice only carries UI state. Selection moves from a per-cell
// `selected: boolean` to a per-table `selectedCells: string[]` array. When a
// table is loaded from a v0 workspace or imported from a legacy export, its
// structural data lands in pendingUpload and is pushed to the server by
// useAutoUpload on first render.
export const stateZ = z.object({
  key: z.string(),
  version: z.literal(VERSION),
  lastSelected: z.string().nullable(),
  editable: z.boolean(),
  selectedCells: z.array(z.string()).default([]),
  pendingUpload: pendingUploadZ.optional(),
});

export interface State extends z.infer<typeof stateZ> {}

export const ZERO_STATE: State = {
  key: "",
  version: VERSION,
  lastSelected: null,
  editable: true,
  selectedCells: [],
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

const buildPendingUpload = (state: v0.State): PendingUpload => {
  const { name: _name, ...rest } = toWire(state, "");
  return rest;
};

// Drops the per-cell selected flag and projects v0's structural model into
// pendingUpload when the table has not yet been synced to the server.
// Workspaces with remoteCreated tables already have the data server-side, so
// migration leaves pendingUpload undefined and the renderer reads from flux.
export const stateMigration = migrate.createMigration<v0.State, State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    key: state.key,
    version: VERSION,
    lastSelected: state.lastSelected,
    editable: state.editable,
    selectedCells: [],
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
