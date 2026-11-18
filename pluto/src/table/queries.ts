// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type project, query, table } from "@synnaxlabs/client";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { deep, grid, id, map, verbs, type xy } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Synnax } from "@/synnax";
import { Cell } from "@/table/cells";
import { Scope } from "@/table/scope";

// The base sizes come from the schema, which declares the default size of a row and a
// column.
const BASE_ROW_SIZE = table.rowZ.parse({}).size;
const BASE_COL_SIZE = table.columnZ.parse({}).size;

const RESOURCE_NAME = "table";

export type RetrieveQuery = table.RetrieveSingleParams;

export const { use, useEnsure, useTombstone, createSelector } = Flux.createRetrieve<
  RetrieveQuery,
  table.Table
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.tables.retrieve(query.key),
  onChange: ({ client, query }, handler) =>
    client.tables.watch(query.key).onChange(handler),
  getCached: ({ client, query }) => client.tables.watch(query.key).getCached(),
  awaitCreation: true,
});

export interface KeyParams {
  key: table.Key;
}

export const useName = Scope.bindHook(createSelector(({ name }) => name));

/** The key, size, and cell keys of a row: what the grid lays out. */
export interface RowLayout {
  key: string;
  size: number;
  cells: string[];
}

/** @returns the layout of each row, without the cell configs. */
export const layoutOf = (rows: table.Row[]): RowLayout[] =>
  rows.map(({ key, size, cells }) => ({ key, size, cells: cells.map((c) => c.key) }));

// Value equality keeps the layout's reference stable across edits that change only a
// cell config.
export const useRows = Scope.bindHook(
  createSelector(({ rows }) => layoutOf(rows), deep.equal),
);

export const useColumns = Scope.bindHook(createSelector(({ columns }) => columns));

export interface CellParams extends KeyParams {
  cellKey: string;
}

export interface RowCellParams extends CellParams {
  /** The key of the row that holds the cell. */
  row: string;
}

const rowOf = (rows: table.Row[], cellKey: string): table.Row | undefined =>
  rows.find((r) => r.cells.some((c) => c.key === cellKey));

const configIn = (row: table.Row | undefined, cellKey: string) =>
  row?.cells.find((c) => c.key === cellKey)?.config;

export const useCell = Scope.bindHook(
  createSelector<Cell.Config | undefined, RowCellParams>(({ rows }, { row, cellKey }) =>
    configIn(
      rows.find((r) => r.key === row),
      cellKey,
    ),
  ),
);

const useRowOf = Scope.bindHook(
  createSelector<string | undefined, CellParams>(
    ({ rows }, { cellKey }) => rowOf(rows, cellKey)?.key,
  ),
);

const useDocumentCellForm = Flux.createDocumentForm<
  table.Key,
  table.Table,
  typeof Cell.configZ,
  RowCellParams
>({
  name: "table cell",
  schema: Cell.configZ,
  domain: (client) => client.tables,
  useEnsure,
  toValues: ({ rows }, { row, cellKey }) =>
    configIn(
      rows.find((r) => r.key === row),
      cellKey,
    ),
  pathOf: ({ row, cellKey }) => ["rows", row, "cells", cellKey, "config"],
});

const useCellFormBase = (query: CellParams) =>
  useDocumentCellForm({ query: { ...query, row: useRowOf(query) ?? "" } });

/** A form over one cell's config in the cached table. */
export const useCellForm = Scope.bindHook(useCellFormBase);

export interface CellsParams extends KeyParams {
  cellKeys: string[];
}

// useCells returns a Map<cellKey, Cell.Config> for the given cellKeys, omitting keys
// that don't resolve to a cell. The map preserves caller-provided key order; consumers
// that need positional iteration should iterate cellKeys and look up via the map.
export const useCells = Scope.bindHook(
  createSelector<Map<string, Cell.Config>, CellsParams>(({ rows }, { cellKeys }) => {
    const configs = new Map<string, Cell.Config>();
    for (const row of rows) for (const c of row.cells) configs.set(c.key, c.config);
    const result = new Map<string, Cell.Config>();
    for (const cellKey of cellKeys) {
      const cell = configs.get(cellKey);
      if (cell != null) result.set(cellKey, cell);
    }
    return result;
  }, map.equal),
);

// useSetChanged returns a function that lists the actions writing only the fields of
// each cell config that differ between base and next. A cell that no row holds yields
// none.
const useSetChangedBase = ({
  key,
}: KeyParams): ((
  changes: Array<Omit<table.SetChangedArgs, "row">>,
) => table.Action[]) => {
  const client = Synnax.use();
  return (changes) => {
    const cached = client?.tables.watch(key).getCached();
    if (!query.isLive(cached)) return [];
    return changes.flatMap((c) => {
      const row = rowOf(cached.rows, c.key);
      return row == null ? [] : table.setChanged({ ...c, row: row.key });
    });
  };
};

export const useSetChanged = Scope.bindHook(useSetChangedBase);

export type DeleteParams = table.Key | table.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.tables.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface CreateParams extends table.New {
  project: project.Key;
}

const createDefaultLayout = (): Pick<table.New, "rows" | "columns"> => {
  const config = Cell.defaultConfig("text");
  const cells = () => [
    { key: id.create(), config },
    { key: id.create(), config },
  ];
  return {
    rows: [
      { size: BASE_ROW_SIZE, cells: cells() },
      { size: BASE_ROW_SIZE, cells: cells() },
    ],
    columns: [{ size: BASE_COL_SIZE }, { size: BASE_COL_SIZE }],
  };
};

const { useUpdate: useCreateBase } = Flux.createUpdate<CreateParams, table.Table>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data, onOptimisticComplete }) =>
    await client.tables.create(data, {
      onOptimistic: async ([optimistic]) => await onOptimisticComplete(optimistic),
    }),
});

const withDefaultLayout = (data: CreateParams): CreateParams =>
  (data.rows?.length ?? 0) === 0 && (data.columns?.length ?? 0) === 0
    ? { ...data, ...createDefaultLayout() }
    : data;

// useCreate creates a new table. If the caller passes no rows or columns,
// the table opens with a 2x2 grid of empty text cells so the user lands on
// a usable starter layout instead of a blank canvas.
export const useCreate: typeof useCreateBase = (args) => {
  const base = useCreateBase(args);
  const baseRef = useSyncedRef(base);
  const update: typeof base.update = (data, opts) =>
    baseRef.current.update(withDefaultLayout(data), opts);
  const updateAsync: typeof base.updateAsync = (data, opts) =>
    baseRef.current.updateAsync(withDefaultLayout(data), opts);
  return { ...base, update, updateAsync };
};

export interface UseRenameParams {
  key: table.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<UseRenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    await client.tables.rename(key, name);
    return data;
  },
});

export const {
  useDispatch,
  useUndo: useUndoBase,
  useRedo: useRedoBase,
  useSingleDispatch: useSingleDispatchBase,
} = Flux.createDispatch<table.Key, table.Table, table.Action>({
  domain: (client) => client.tables,
});

export const useSingleDispatch = Scope.bindHook(useSingleDispatchBase);
export const useUndo = Scope.bindHook(useUndoBase);
export const useRedo = Scope.bindHook(useRedoBase);

// findCellPosition returns the (x, y) grid coordinates of the cell with the given key
// in the given rows, or null if the cell isn't referenced by any row.
export const findCellPosition = (rows: RowLayout[], cellKey: string): xy.XY | null => {
  for (let y = 0; y < rows.length; y++) {
    const x = rows[y].cells.indexOf(cellKey);
    if (x !== -1) return { x, y };
  }
  return null;
};

// nextCellPosition returns the (x, y) grid coordinates of the cell one step forward
// (dir=1) or backward (dir=-1) from pos in row-major order. At a row's end (forward)
// the position wraps to the start of the next row; (backward) to the end of the
// previous row. Returns null when the move would step past the first or last cell of
// the table, when pos's row is out of range, or when the resulting row has no cells
// (e.g. a ragged asymmetric state).
export const nextCellPosition = (
  rows: RowLayout[],
  pos: xy.XY,
  dir: 1 | -1,
): xy.XY | null => {
  if (rows[pos.y] == null) return null;
  const x = pos.x + dir;
  if (x >= 0 && x < rows[pos.y].cells.length) return { x, y: pos.y };
  const y = pos.y + dir;
  if (y < 0 || y >= rows.length) return null;
  const row = rows[y];
  if (row.cells.length === 0) return null;
  return { x: dir === 1 ? 0 : row.cells.length - 1, y };
};

// cellsInRegion returns the cell keys inside the inclusive axis-aligned
// rectangle defined by start and end. Rows or row slots outside the rectangle
// are skipped silently; ragged rows produce a sparse region.
export const cellsInRegion = (rows: RowLayout[], start: xy.XY, end: xy.XY): string[] =>
  grid
    .region(start, end)
    .map(({ x, y }) => rows[y]?.cells[x])
    .filter((cellKey) => cellKey != null);

const useCellPositionBase = ({ key, cellKey }: CellParams): xy.XY | null => {
  const rows = useRows({ key });
  return findCellPosition(rows, cellKey);
};

export const useCellPosition = Scope.bindHook(useCellPositionBase);
