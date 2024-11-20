import * as v0 from "@/table/migrations/v0";

export type State = v0.State;
export type SliceState = v0.SliceState;
export type CellState<S extends string = string, P = unknown> = v0.CellState<S, P>;
export type RowLayout = v0.RowLayout;
export type CellLayout = v0.CellLayout;
export const ZERO_STATE = v0.ZERO_STATE;
export const ZERO_SLICE_STATE = v0.ZERO_SLICE_STATE;
export const ZERO_CELL_STATE = v0.ZERO_CELL_STATE;
export const ZERO_CELL_PROPS = v0.ZERO_TEXT_CELL_PROPS;
