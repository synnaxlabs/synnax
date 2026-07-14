// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type project, table } from "@synnaxlabs/client";
import { array, compare, id, uuid, type xy } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";

import { Flux } from "@/flux";
import { useSyncedRef } from "@/hooks/ref";
import { Ontology } from "@/ontology";
import { Cell } from "@/table/cells";

const BASE_ROW_SIZE = 36;
const BASE_COL_SIZE = 72;

export const FLUX_STORE_KEY = "tables";
const RESOURCE_NAME = "table";

export interface FluxStore extends Flux.UndoableUnaryStore<
  table.Key,
  table.Table,
  table.Action
> {}

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

export type RetrieveQuery = table.RetrieveSingleParams;

export const retrieveSingle = async ({
  store,
  client,
  query: { key },
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const cached = store.tables.get(key);
  if (cached != null) return cached;
  const t = await client.tables.retrieve({ key });
  store.tables.set(t);
  return t;
};

export const { useRetrieve, useRetrieveObservable, useEnsureRetrieved } =
  Flux.createRetrieve<RetrieveQuery, table.Table, FluxSubStore>({
    name: RESOURCE_NAME,
    retrieve: retrieveSingle,
    mountListeners: ({ store, query: { key }, onChange }) =>
      store.tables.onSet(onChange, key),
  });

export const useRetrieveObservableName = ({
  onChange,
  ...params
}: Omit<Flux.UseRetrieveObservableParams<RetrieveQuery, table.Table>, "onChange"> & {
  onChange: (name: string) => void;
}): Flux.UseRetrieveObservableReturn<RetrieveQuery> => {
  const onChangeRef = useSyncedRef(onChange);
  return useRetrieveObservable({
    ...params,
    onChange: useCallback((result) => {
      if (result.variant !== "success") return;
      onChangeRef.current(result.data.name);
    }, []),
  });
};

export interface SelectKeyArgs {
  key: table.Key;
}

const requireTable = (store: FluxSubStore, key: table.Key): table.Table => {
  const t = store.tables.get(key);
  if (t == null) throw new NotFoundError(`Table with key ${key} not found`);
  return t;
};

export const useSelectName = Flux.createSelector<FluxSubStore, SelectKeyArgs, string>({
  subscribe: (store, { key }, notify) => store.tables.onSet(notify, key),
  select: (store, { key }) => requireTable(store, key).name,
});

export const useSelectRows = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  table.Row[]
>({
  subscribe: (store, { key }, notify) => store.tables.onSet(notify, key),
  select: (store, { key }) => requireTable(store, key).rows,
});

export const useSelectColumns = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  table.Column[]
>({
  subscribe: (store, { key }, notify) => store.tables.onSet(notify, key),
  select: (store, { key }) => requireTable(store, key).columns,
});

export interface SelectCellArgs {
  key: table.Key;
  cellKey: string;
}

export const useSelectCell = Flux.createSelector<
  FluxSubStore,
  SelectCellArgs,
  Cell.Config | undefined
>({
  subscribe: (store, { key }, notify) => store.tables.onSet(notify, key),
  select: (store, { key, cellKey }) => store.tables.get(key)?.cells?.[cellKey],
});

export interface SelectCellsArgs {
  key: table.Key;
  cellKeys: string[];
}

// useSelectCells returns a Map<cellKey, Cell.Config> for the given cellKeys,
// omitting keys that don't resolve to a cell. The map preserves
// caller-provided key order; consumers that need positional iteration should
// iterate cellKeys and look up via the map.
export const useSelectCells = Flux.createSelector<
  FluxSubStore,
  SelectCellsArgs,
  Map<string, Cell.Config>
>({
  subscribe: (store, { key }, notify) => store.tables.onSet(notify, key),
  select: (store, { key, cellKeys }) => {
    const result = new Map<string, Cell.Config>();
    if (cellKeys.length === 0) return result;
    const t = store.tables.get(key);
    if (t == null) return result;
    for (const cellKey of cellKeys) {
      const cell = t.cells?.[cellKey];
      if (cell != null) result.set(cellKey, cell);
    }
    return result;
  },
  equal: compare.mapsEqual,
});

export type DeleteParams = table.Key | table.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const keys = array.toArray(data);
    const ids = table.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    await client.tables.delete(data);
    return data;
  },
});

export interface CreateParams extends table.New {
  project?: project.Key;
}

const createDefaultLayout = (): Pick<table.Table, "rows" | "columns" | "cells"> => {
  const cellKeys = [id.create(), id.create(), id.create(), id.create()];
  const config = Cell.defaultConfig("text");
  return {
    rows: [
      { size: BASE_ROW_SIZE, cells: [cellKeys[0], cellKeys[1]] },
      { size: BASE_ROW_SIZE, cells: [cellKeys[2], cellKeys[3]] },
    ],
    columns: [{ size: BASE_COL_SIZE }, { size: BASE_COL_SIZE }],
    cells: Object.fromEntries(cellKeys.map((k) => [k, config])),
  };
};

const { useUpdate: useCreateBase } = Flux.createUpdate<
  CreateParams,
  FluxSubStore,
  table.Table
>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const optimistic = table.newZ.parse(data);
    rollbacks.push(store.tables.set(optimistic));
    const project = data.project ?? uuid.ZERO;
    const created = await client.tables.create(project, optimistic);
    store.tables.set(created);
    return created;
  },
});

// useCreate creates a new table. If the caller passes no rows or columns,
// the table opens with a 2x2 grid of empty text cells so the user lands on
// a usable starter layout instead of a blank canvas.
export const useCreate: typeof useCreateBase = (args) => {
  const base = useCreateBase(args);
  const baseRef = useSyncedRef(base);
  const withDefaultLayout = useCallback(
    (data: CreateParams): CreateParams =>
      (data.rows?.length ?? 0) === 0 && (data.columns?.length ?? 0) === 0
        ? { ...data, ...createDefaultLayout() }
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

export interface UseRenameArgs {
  key: table.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<UseRenameArgs, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const { key, name } = data;
    rollbacks.push(Flux.partialUpdate(store.tables, key, { name }));
    rollbacks.push(Ontology.renameFluxResource(store, table.ontologyID(key), name));
    await client.tables.rename(key, name);
    return data;
  },
});

// kindOfTransaction classifies an action batch for the undoable store's
// per-kind coalesce window. Continuous resize gestures (a stream of
// resize_row or resize_col actions) collapse into a single undo step;
// successive set_cell actions on the same cell coalesce per-cell so typing
// inside one cell collapses to one undo step but switching cells starts a
// fresh entry; everything else is treated as a discrete user action.
const kindOfTransaction = (actions: table.Action[]): string => {
  if (actions.length === 0) return "default";
  if (actions.every((a) => a.type === "resize_row" || a.type === "resize_col"))
    return "resize";
  if (actions.length === 1) {
    const a = actions[0];
    if (a.type === "set_cell") return `set_cell:${a.setCell.cell.key}`;
    return a.type;
  }
  return "transaction";
};

export const FLUX_STORE_CONFIG = Flux.createUndoableStore<
  table.Key,
  table.Table,
  table.Action,
  typeof FLUX_STORE_KEY,
  FluxSubStore
>({
  storeKey: FLUX_STORE_KEY,
  reduce: table.reduceAll,
  channel: table.SET_CHANNEL_NAME,
  schema: table.scopedActionZ,
  kindOf: kindOfTransaction,
});

export const { useDispatch, useUndo, useRedo } = Flux.createDispatch<
  table.Key,
  table.Table,
  table.Action,
  typeof FLUX_STORE_KEY,
  FluxSubStore
>({
  storeKey: FLUX_STORE_KEY,
  send: ({ client, key, actions, dispatchKey }) =>
    client.tables.dispatch(key, dispatchKey, actions),
});

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

export const useCellPosition = ({ key, cellKey }: SelectCellArgs): xy.XY | null => {
  const rows = useSelectRows({ key });
  return useMemo(() => findCellPosition(rows, cellKey), [rows, cellKey]);
};
