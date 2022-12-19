import { createSlice } from "@reduxjs/toolkit";
import type { PayloadAction } from "@reduxjs/toolkit";

import { Range } from "./types";

export interface WorkspaceState {
  ranges: Range[];
}

export interface WorkspaceStoreState {
  workspace: WorkspaceState;
}

export const initialState: WorkspaceState = {
  ranges: [
    {
      key: "range1",
      name: "Range 1",
      start: 125261234614689,
      end: 225261234614699,
    },
  ],
};

type AddRangeAction = PayloadAction<Range>;
type RemoveRangeAction = PayloadAction<string>;

export const {
  actions: { addRange, removeRange },
  reducer: workspaceReducer,
} = createSlice({
  name: "workspace",
  initialState,
  reducers: {
    addRange: (state, { payload }: AddRangeAction) => {
      state.ranges.push(payload);
    },
    removeRange: (state, { payload }: RemoveRangeAction) => {
      state.ranges = state.ranges.filter((range) => range.key !== payload);
    },
  },
});
