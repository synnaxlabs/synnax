// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  NotFoundError,
  type project,
  query,
  type Synnax as Client,
  type table,
} from "@synnaxlabs/client";
import { compare, id, uuid, type xy } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";

import { Flux } from "@/flux";
import { useSyncedRef } from "@/hooks/ref";
import { Cell } from "@/table/cells";
import { Scope } from "@/table/scope";
import { Theming } from "@/theming";

const BASE_ROW_SIZE = 36;
const BASE_COL_SIZE = 72;

const RESOURCE_NAME = "table";

export type RetrieveQuery = table.RetrieveSingleParams;

export const { useRetrieve, useRetrieveObservable, useEnsureRetrieved } =
  Flux.createRetrieve<RetrieveQuery, table.Table>({
    name: RESOURCE_NAME,
    retrieve: async ({ client, query }) => await client.tables.retrieve(query),
    subscribe: ({ client, query }, handler) => client.tables.onChange(query, handler),
    getCached: ({ client, query }) => client.tables.getCached(query),
  });

export interface SelectKeyParams {
  key: table.Key;
}

const requireTable = (client: Client | null, key: table.Key): table.Table => {
  const cached = client?.tables.getCached({ key });
  if (cached == null) throw new NotFoundError(`Table with key ${key} not found`);
  if (query.Deleted.matches(cached))
    throw new Flux.DeletedError(`${RESOURCE_NAME} was deleted`, cached.corpse);
  return cached;
};

const getTable = (client: Client | null, key: table.Key): table.Table | undefined => {
  const cached = client?.tables.getCached({ key });
  if (!query.isLive(cached)) return undefined;
  return cached;
};

const subscribe = (
  { client, args: { key } }: Flux.SelectorParams<SelectKeyParams>,
  notify: () => void,
) => (client == null ? () => {} : client.tables.onChange({ key }, notify));

export const [useSelectName, useGetName] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, string>({
    subscribe,
    select: ({ client, args: { key } }) => requireTable(client, key).name,
  }),
);

export const [useSelectRows, useGetRows] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, table.Row[]>({
    subscribe,
    select: ({ client, args: { key } }) => requireTable(client, key).rows,
  }),
);

export const [useSelectColumns, useGetColumns] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, table.Column[]>({
    subscribe,
    select: ({ client, args: { key } }) => requireTable(client, key).columns,
  }),
);

export interface SelectCellParams {
  key: table.Key;
  cellKey: string;
}

export const [useSelectCell, useGetCell] = Scope.bindSelector(
  Flux.createSelector<SelectCellParams, Cell.Config | undefined>({
    subscribe,
    select: ({ client, args: { key, cellKey } }) =>
      getTable(client, key)?.cells?.[cellKey] as Cell.Config | undefined,
  }),
);

export interface SelectCellsParams {
  key: table.Key;
  cellKeys: string[];
}

// useSelectCells returns a Map<cellKey, Cell.Config> for the given cellKeys,
// omitting keys that don't resolve to a cell. The map preserves
// caller-provided key order; consumers that need positional iteration should
// iterate cellKeys and look up via the map.
export const [useSelectCells, useGetCells] = Scope.bindSelector(
  Flux.createSelector<SelectCellsParams, Map<string, Cell.Config>>({
    subscribe,
    select: ({ client, args: { key, cellKeys } }) => {
      const result = new Map<string, Cell.Config>();
      const t = getTable(client, key);
      if (t == null || cellKeys.length === 0) return result;
      for (const cellKey of cellKeys) {
        const cell = t.cells?.[cellKey] as Cell.Config | undefined;
        if (cell != null) result.set(cellKey, cell);
      }
      return result;
    },
    equal: compare.mapsEqual,
  }),
);

export type DeleteParams = table.Key | table.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.tables.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface CreateParams extends table.New {
  project?: project.Key;
}

const createDefaultLayout = (
  theme: ReturnType<typeof Theming.use>,
): Pick<table.Table, "rows" | "columns" | "cells"> => {
  const cellKeys = [id.create(), id.create(), id.create(), id.create()];
  const props = Cell.REGISTRY.text.defaultProps(theme);
  return {
    rows: [
      { size: BASE_ROW_SIZE, cells: [cellKeys[0], cellKeys[1]] },
      { size: BASE_ROW_SIZE, cells: [cellKeys[2], cellKeys[3]] },
    ],
    columns: [{ size: BASE_COL_SIZE }, { size: BASE_COL_SIZE }],
    cells: Object.fromEntries(
      cellKeys.map((k) => [k, { key: k, variant: "text", props }]),
    ),
  };
};

const { useUpdate: useCreateBase } = Flux.createUpdate<CreateParams, table.Table>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, onOptimisticComplete }) =>
    await client.tables.create(data.project ?? uuid.ZERO, data, {
      onOptimistic: async ([optimistic]) => await onOptimisticComplete(optimistic),
    }),
});

// useCreate creates a new table. If the caller passes no rows or columns,
// the table opens with a 2x2 grid of empty text cells so the user lands on
// a usable starter layout instead of a blank canvas.
export const useCreate: typeof useCreateBase = (args) => {
  const base = useCreateBase(args);
  const baseRef = useSyncedRef(base);
  const themeRef = useSyncedRef(Theming.use());
  const withDefaultLayout = useCallback(
    (data: CreateParams): CreateParams =>
      (data.rows?.length ?? 0) === 0 && (data.columns?.length ?? 0) === 0
        ? { ...data, ...createDefaultLayout(themeRef.current) }
        : data,
    [],
  );
  const update = useCallback<typeof base.update>(
    (data, opts) => baseRef.current.update(withDefaultLayout(data), opts),
    [withDefaultLayout],
  );
  const updateAsync = useCallback<typeof base.updateAsync>(
    (data, opts) => baseRef.current.updateAsync(withDefaultLayout(data), opts),
    [withDefaultLayout],
  );
  return { ...base, update, updateAsync };
};

export interface UseRenameParams {
  key: table.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<UseRenameParams>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await onOptimisticComplete(data);
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

// findCellPosition returns the (x, y) grid coordinates of the cell with the
// given key in the given rows, or null if the cell isn't referenced by any
// row.
export const findCellPosition = (rows: table.Row[], cellKey: string): xy.XY | null => {
  for (let y = 0; y < rows.length; y++) {
    const x = rows[y].cells.indexOf(cellKey);
    if (x !== -1) return { x, y };
  }
  return null;
};

// nextCellPosition returns the (x, y) grid coordinates of the cell one step
// forward (dir=1) or backward (dir=-1) from pos in row-major order. At a
// row's end (forward) the position wraps to the start of the next row;
// (backward) to the end of the previous row. Returns null when the move
// would step past the first or last cell of the table, when pos's row is
// out of range, or when the resulting row has no cells (e.g. a ragged
// asymmetric state).
export const nextCellPosition = (
  rows: table.Row[],
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
export const cellsInRegion = (
  rows: table.Row[],
  start: xy.XY,
  end: xy.XY,
): string[] => {
  const minX = Math.min(start.x, end.x);
  const maxX = Math.max(start.x, end.x);
  const minY = Math.min(start.y, end.y);
  const maxY = Math.max(start.y, end.y);
  const out: string[] = [];
  for (let y = minY; y <= maxY; y++) {
    const row = rows[y];
    if (row == null) continue;
    for (let x = minX; x <= maxX; x++) if (row.cells[x] != null) out.push(row.cells[x]);
  }
  return out;
};

export const useCellPosition = Scope.bindHook(
  ({ key, cellKey }: SelectCellParams): xy.XY | null => {
    const rows = useSelectRows({ key });
    return useMemo(() => findCellPosition(rows, cellKey), [rows, cellKey]);
  },
);
