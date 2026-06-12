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

// segProp reads a property from a named segment of a legacy pipeline spec;
// single-segment pipelines store the spec at the top level. Mirrors
// convert.go on the server side.
const segProp = (spec: unknown, segment: string, prop: string): unknown => {
  if (typeof spec !== "object" || spec == null) return undefined;
  const props = (spec as record.Unknown).props;
  if (typeof props !== "object" || props == null) return undefined;
  let segments = (props as record.Unknown).segments;
  if (typeof segments !== "object" || segments == null) segments = { [segment]: spec };
  const seg = (segments as record.Unknown)[segment];
  if (typeof seg !== "object" || seg == null) return undefined;
  const segProps = (seg as record.Unknown).props;
  if (typeof segProps !== "object" || segProps == null) return undefined;
  const v = (segProps as record.Unknown)[prop];
  if (prop === "channel" && v === 0) return undefined;
  return v;
};

// LEGACY_ALIGNS maps the x-location alignments the pre-typed text cell schema
// declared onto the flex alignments the alignment form control has always
// written.
const LEGACY_ALIGNS: Record<string, string> = { left: "start", right: "end" };

// LEGACY_WEIGHTS maps named CSS font weights onto the numeric weights the
// weight form control writes.
const LEGACY_WEIGHTS: Record<string, number> = {
  lighter: 300,
  normal: 400,
  bold: 700,
  bolder: 700,
};

// extractLegacyArgs rewrites a legacy cell's stored fields into the semantic
// arguments the typed schema declares, in place. Mirrors convert.go.
const extractLegacyArgs = (cfg: record.Unknown): void => {
  const setIfPresent = (key: string, v: unknown): void => {
    if (v !== undefined) cfg[key] = v;
  };
  switch (cfg.variant) {
    case "value":
      setIfPresent("channel", segProp(cfg.telem, "valueStream", "channel"));
      setIfPresent(
        "rollingAverage",
        segProp(cfg.telem, "rollingAverage", "windowSize"),
      );
      setIfPresent("precision", segProp(cfg.telem, "stringifier", "precision"));
      setIfPresent("notation", segProp(cfg.telem, "stringifier", "notation"));
      delete cfg.telem;
      break;
    case "text":
      if (typeof cfg.align === "string" && cfg.align in LEGACY_ALIGNS)
        cfg.align = LEGACY_ALIGNS[cfg.align];
      if (typeof cfg.weight === "string" && cfg.weight in LEGACY_WEIGHTS)
        cfg.weight = LEGACY_WEIGHTS[cfg.weight];
      break;
  }
};

const EMPTY_TEXT_CONFIG: table.CellConfig = { variant: "text" };

// migrateCell converts a legacy cell (variant string plus camelCase props
// written verbatim) into the typed cell config. Cells that conform to no
// known variant degrade to an empty text cell so rows never reference missing
// entries.
const migrateCell = (c: { variant: string; props?: unknown }): table.CellConfig => {
  const cfg: record.Unknown = {
    ...((c.props as record.Unknown) ?? {}),
    variant: c.variant,
  };
  extractLegacyArgs(cfg);
  const parsed = table.cellConfigZ.safeParse(cfg);
  return parsed.success ? parsed.data : EMPTY_TEXT_CONFIG;
};

const buildPendingUpload = (state: v0.State): PendingUpload => ({
  key: state.key,
  rows: state.layout.rows.map((r) => ({
    size: r.size,
    cells: r.cells.map((c) => c.key),
  })),
  columns: state.layout.columns,
  cells: Object.fromEntries(
    Object.entries(state.cells).map(([k, c]) => [k, migrateCell(c)]),
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
