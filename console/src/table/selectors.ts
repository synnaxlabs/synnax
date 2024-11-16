import { useMemoSelect } from "@/hooks";
import {
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

export const selectCell = (state: StoreState, key: string, cellKey: string) =>
  select(state, key).cells[cellKey];

export const useSelectCell = (key: string, cellKey: string) =>
  useMemoSelect((state: StoreState) => selectCell(state, key, cellKey), [key, cellKey]);

export const selectLayout = (state: StoreState, key: string) =>
  select(state, key).layout;

export const useSelectLayout = (key: string) =>
  useMemoSelect((state: StoreState) => selectLayout(state, key), [key]);
