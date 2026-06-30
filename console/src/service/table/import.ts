// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, table } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { migrate, type record } from "@synnaxlabs/x";
import { z } from "zod";

import { type Import } from "@/import";
import { create, LAYOUT_TYPE } from "@/component/table/layout";

const STATE_MIGRATION_NAME = "table.state";

const V0_VERSION = "0.0.0";

// These schemas are a frozen snapshot of the layout/cell shapes that shipped at
// state version 0.0.0, when the structural model still lived in console state.
const v0CellLayoutZ = z.object({ key: z.string() });
const v0RowLayoutZ = z.object({ size: z.number(), cells: z.array(v0CellLayoutZ) });
const v0ColLayoutZ = z.object({ size: z.number() });
const v0CellStateZ = z.object({
  key: z.string(),
  variant: z.string(),
  selected: z.boolean(),
  props: z.unknown(),
});

const v0StateZ = z.object({
  key: z.string(),
  version: z.literal(V0_VERSION),
  lastSelected: z.string().nullable(),
  editable: z.boolean(),
  layout: z.object({ rows: z.array(v0RowLayoutZ), columns: v0ColLayoutZ.array() }),
  cells: z.record(z.string(), v0CellStateZ),
  remoteCreated: z.boolean(),
});
interface V0State extends z.infer<typeof v0StateZ> {}

const V1_VERSION = "1.0.0";

// pendingUploadZ is the table payload needed to upload a legacy v0 table. name
// is omitted because the live name lives in Layout.
const pendingUploadZ = table.tableZ.omit({ name: true });
interface PendingUpload extends z.infer<typeof pendingUploadZ> {}

const v1StateZ = z.object({
  key: z.string(),
  version: z.literal(V1_VERSION),
  lastSelected: z.string().nullable(),
  editable: z.boolean(),
  selectedCells: z.array(z.string()).default([]),
  hideIndicators: z.boolean().default(false),
  pendingUpload: pendingUploadZ.optional(),
});
interface V1State extends z.infer<typeof v1StateZ> {}

const ZERO_STATE: V1State = {
  key: "",
  version: V1_VERSION,
  lastSelected: null,
  editable: true,
  selectedCells: [],
  hideIndicators: false,
  pendingUpload: undefined,
};

const buildPendingUpload = (state: V0State): PendingUpload => ({
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

// v1StateMigration drops the per-cell selected flag and parks v0's structural
// model in pendingUpload when the table is not yet remoteCreated. remoteCreated
// tables already have authoritative data on the server, so pendingUpload stays
// undefined for those.
const v1StateMigration = migrate.createMigration<V0State, V1State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    key: state.key,
    version: V1_VERSION,
    lastSelected: state.lastSelected,
    editable: state.editable,
    selectedCells: [],
    hideIndicators: false,
    pendingUpload: state.remoteCreated ? undefined : buildPendingUpload(state),
  }),
});

type AnyState = V0State | V1State;

const STATE_MIGRATIONS: migrate.Migrations = {
  [V0_VERSION]: v1StateMigration,
};

export const migrateState = migrate.migrator<AnyState, V1State>({
  name: STATE_MIGRATION_NAME,
  migrations: STATE_MIGRATIONS,
  def: ZERO_STATE,
});

// anyStateZ parses either a v0 or v1 state blob and returns a uniform v1 state.
// v0 inputs flow through the migration ladder, so pendingUpload is populated
// when the source carried unsynced structural data.
export const anyStateZ = z.union([v1StateZ, v0StateZ]).transform(migrateState);

export const parseImport = (
  data: unknown,
  fallbackName: string | undefined,
): table.New => {
  // Legacy console-state exports are tried first because their schemas are strict: they
  // require a version literal plus the console-only lastSelected / editable / layout
  // fields, so a current typed export never matches and falls through to the direct
  // branch. The typed tableZ is the opposite — it strips unknown keys and defaults rows
  // / columns / cells to empty, so trying it first silently accepts a legacy file,
  // drops its structural model, and yields a table with no rows or columns (a blank
  // import).
  if (typeof data === "object" && data != null) {
    const legacy = anyStateZ.safeParse({ ...data, remoteCreated: false });
    if (legacy.success) {
      if (legacy.data.pendingUpload == null)
        throw new Error("Imported table has no structural data");
      const { key: _key, rows, columns, cells } = legacy.data.pendingUpload;
      return { name: fallbackName ?? "Table", rows, columns, cells };
    }
  }
  const { key: _key, ...rest } = table.tableZ.parse(data);
  return { ...rest, name: fallbackName ?? rest.name };
};

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client, projectKey },
) => {
  if (!Access.updateGranted({ id: table.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import tables");
  if (client == null) throw new DisconnectedError();
  const newPayload = parseImport(data, layout?.name);
  const created = await client.tables.create(projectKey, newPayload);
  store.tables.set(created.key, created);
  placeLayout(
    create({ ...layout, key: created.key, name: created.name, type: LAYOUT_TYPE }),
  );
  return table.ontologyID(created.key);
};

export const FILE_INGESTERS: Import.FileIngesters = { [LAYOUT_TYPE]: ingest };
