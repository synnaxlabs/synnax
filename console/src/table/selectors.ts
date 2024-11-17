import { i } from "node_modules/vite/dist/node/types.d-aGj9QkWt";

import { useMemoSelect } from "@/hooks";
import {
  type CellState,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
} from "@/table/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).tables[key];

export const useSelect = (key: string): State =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const selectCell = <T extends string = string, P = unknown>(
  state: StoreState,
  key: string,
  cellKey: string,
): CellState<T, P> => select(state, key).cells[cellKey] as CellState<T, P>;

export const useSelectCell = <T extends string = string, P = unknown>(
  key: string,
  cellKey: string,
): CellState<T, P> =>
  useMemoSelect(
    (state: StoreState) => selectCell<T, P>(state, key, cellKey),
    [key, cellKey],
  );

export const selectLayout = (state: StoreState, key: string) =>
  select(state, key).layout;

export const useSelectLayout = (key: string) =>
  useMemoSelect((state: StoreState) => selectLayout(state, key), [key]);

const selectSelectedCells = <T extends string, P = unknown>(
  state: StoreState,
  key: string,
): CellState<T, P>[] =>
  Object.values(select(state, key).cells).filter((cell) => cell.selected) as CellState<
    T,
    P
  >[];

export const useSelectSelectedCells = <T extends string = string, P = unknown>(
  key: string,
): CellState<T, P>[] =>
  useMemoSelect((state: StoreState) => selectSelectedCells<T, P>(state, key), [key]);

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
  console.log(columns);
  return Array.from(columns);
};

export const useSelectSelectedColumns = (key: string): number[] =>
  useMemoSelect((state: StoreState) => selectSelectedColumns(state, key), [key]);
