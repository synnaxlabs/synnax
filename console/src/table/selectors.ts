// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";

import { useMemoSelect } from "@/hooks";
import {
  SLICE_NAME,
  type State,
  type SliceState,
  type StoreState,
  type ToolbarState,
  type CellState,
} from "@/table/slice";

export const selectSlice = (state: StoreState): SliceState => state[SLICE_NAME];

export const select = (state: StoreState, key: string): State =>
  selectSlice(state).tables[key];

export const useSelect = (key: string): State =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const selectToolbar = (state: StoreState): ToolbarState =>
  selectSlice(state).toolbar;

export const useSelectToolbar = (): ToolbarState => useMemoSelect(selectToolbar, []);

export const selectSelected = (state: StoreState, key: string): xy.XY[] =>
  select(state, key).selected;

export interface SugaredCellstate extends CellState {
  pos: xy.XY;
}

export const selectSelectedCells = (
  state: StoreState,
  key: string,
): SugaredCellstate[] => {
  const table = select(state, key);
  return table.selected.map((pos) => ({ ...table.rows[pos.y].cells[pos.x], pos }));
};

export const useSelectSelectedCells = (key: string): SugaredCellstate[] =>
  useMemoSelect((state: StoreState) => selectSelectedCells(state, key), [key]);
