// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type TableCells } from "@synnaxlabs/pluto";
import { type record, type xy } from "@synnaxlabs/x";

import { useMemoSelect } from "@/hooks";
import {
  type CellState,
  findCellPosition,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
} from "@/table/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).tables[key];

export const selectOptional = select as (
  state: StoreState,
  key: string,
) => State | undefined;

export const useSelect = (key: string): State =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const selectCell = <
  V extends TableCells.Variant = TableCells.Variant,
  P extends object = record.Unknown,
>(
  state: StoreState,
  key: string,
  cellKey: string,
): CellState<V, P> => select(state, key).cells[cellKey] as CellState<V, P>;

export const useSelectCell = <
  V extends TableCells.Variant = TableCells.Variant,
  P extends object = record.Unknown,
>(
  key: string,
  cellKey: string,
): CellState<V, P> =>
  useMemoSelect(
    (state: StoreState) => selectCell<V, P>(state, key, cellKey),
    [key, cellKey],
  );

export const selectCellType = <V extends TableCells.Variant = TableCells.Variant>(
  state: StoreState,
  key: string,
  cellKey: string,
): V => selectCell(state, key, cellKey).variant as V;

export const useSelectCellType = <V extends TableCells.Variant = TableCells.Variant>(
  key: string,
  cellKey: string,
): V =>
  useMemoSelect(
    (state: StoreState) => selectCellType<V>(state, key, cellKey),
    [key, cellKey],
  );

export const selectLayout = (state: StoreState, key: string) =>
  select(state, key).layout;

export const useSelectLayout = (key: string) =>
  useMemoSelect((state: StoreState) => selectLayout(state, key), [key]);

const selectSelectedCells = <
  V extends TableCells.Variant = TableCells.Variant,
  P extends object = record.Unknown,
>(
  state: StoreState,
  key: string,
): CellState<V, P>[] => {
  const table = selectOptional(state, key);
  if (table == null) return [];
  return Object.values(table.cells).filter((cell) => cell.selected) as CellState<
    V,
    P
  >[];
};

export const useSelectSelectedCells = <
  V extends TableCells.Variant = TableCells.Variant,
  P extends object = record.Unknown,
>(
  key: string,
): CellState<V, P>[] =>
  useMemoSelect((state: StoreState) => selectSelectedCells<V, P>(state, key), [key]);

export const selectSelectedColumns = (state: StoreState, key: string): number[] => {
  const table = select(state, key);
  const selected = selectSelectedCells(state, key);
  const selectedKeys = selected.map((cell) => cell.key);
  const columns = new Set<number>();
  table.layout.rows.forEach((row) => {
    row.cells.forEach((cell, j) => {
      if (selectedKeys.includes(cell.key)) columns.add(j);
    });
  });
  return Array.from(columns);
};

export const useSelectSelectedColumns = (key: string): number[] =>
  useMemoSelect((state: StoreState) => selectSelectedColumns(state, key), [key]);

export const selectSellLocation = (state: StoreState, key: string): xy.XY | null => {
  const table = select(state, key);
  return findCellPosition(table, key);
};

export const useSelectSellLocation = (key: string): xy.XY | null =>
  useMemoSelect((state: StoreState) => selectSellLocation(state, key), [key]);

export const selectSelectedCellPos = (state: StoreState, key: string): xy.XY | null => {
  const table = select(state, key);
  const selected = selectSelectedCells(state, key);
  if (selected.length === 0) return null;
  const pos = findCellPosition(table, selected[0].key);
  return pos;
};

export const useSelectSelectedCellPos = (key: string): xy.XY | null =>
  useMemoSelect((state: StoreState) => selectSelectedCellPos(state, key), [key]);

export const selectEditable = (state: StoreState, key: string): boolean =>
  select(state, key).editable;

export const useSelectEditable = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectEditable(state, key), [key]);

export const selectVersion = (state: StoreState, key: string): string | undefined =>
  selectOptional(state, key)?.version;

export const useSelectVersion = (key: string): string | undefined =>
  useMemoSelect((state: StoreState) => selectVersion(state, key), [key]);
