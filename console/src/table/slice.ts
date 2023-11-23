// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { TableCell, Theming } from "@synnaxlabs/pluto";
import { type dimensions, type xy } from "@synnaxlabs/x";
import { v4 as uuidV4 } from "uuid";

import { type Layout } from "@/layout";

export interface State {
  rows: RowState[];
  dimensions: dimensions.Dimensions;
  selected: xy.XY[];
}
export interface CellState {
  type: string;
  props: Record<string, any>;
}

const ZERO_CELL_STATE: CellState = {
  type: TableCell.REGISTRY.label.type,
  props: TableCell.REGISTRY.label.initialProps(Theming.themes.synnaxDark),
};

const TOOLBAR_TABS = ["shape", "properties"] as const;
export type ToolbarTab = (typeof TOOLBAR_TABS)[number];

export interface ToolbarState {
  activeTab: ToolbarTab;
}

export interface RowState {
  cells: CellState[];
}

const zeroRowState = (width: number): RowState => ({
  cells: Array.from({ length: width }, () => ({ ...ZERO_CELL_STATE })),
});

export interface SliceState {
  toolbar: ToolbarState;
  tables: Record<string, State>;
}

export const SLICE_NAME = "table";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_STATE: State = {
  dimensions: { width: 4, height: 4 },
  rows: [zeroRowState(4), zeroRowState(4), zeroRowState(4), zeroRowState(4)],
  selected: [],
};

export const ZERO_SLICE_STATE: SliceState = {
  toolbar: {
    activeTab: "shape",
  },
  tables: {},
};

export interface SetActiveToolbarTabPayload {
  tab: ToolbarTab;
}

export interface SetCellPropsPayload {
  key: string;
  positions: xy.XY[];
  props: Array<Record<string, any>>;
}

export interface CreatePayload extends State {
  key: string;
}

export interface SetSelectedPayload {
  key: string;
  selected: xy.XY[];
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setActiveToolbarTab: (state, action: PayloadAction<SetActiveToolbarTabPayload>) => {
      state.toolbar.activeTab = action.payload.tab;
    },
    create: (state, action: PayloadAction<CreatePayload>) => {
      state.tables[action.payload.key] = action.payload;
    },
    setSelected: (state, action: PayloadAction<SetSelectedPayload>) => {
      state.tables[action.payload.key].selected = action.payload.selected;
    },
    setCellProps: (
      state,
      { payload: { key, positions, props } }: PayloadAction<SetCellPropsPayload>,
    ) => {
      const t = state.tables[key];
      positions.forEach((pos, i) => {
        const cell = t.rows[pos.y].cells[pos.x];
        cell.props = { ...cell.props, ...props[i] };
      });
    },
  },
});

export const { setSelected, setActiveToolbarTab, setCellProps } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];

export type LayoutType = "table";
export const LAYOUT_TYPE = "table";

export const create =
  (
    initial: Partial<State> & Omit<Partial<Layout.LayoutState>, "type">,
  ): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Table", location = "mosaic", window, tab, ...rest } = initial;
    const key = initial.key ?? uuidV4();
    dispatch(actions.create({ ...ZERO_STATE, ...rest, key }));
    return {
      name,
      location,
      window,
      tab,
      key,
      type: LAYOUT_TYPE,
    };
  };
