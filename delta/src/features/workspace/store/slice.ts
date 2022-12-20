import { createSlice } from "@reduxjs/toolkit";
import type { PayloadAction } from "@reduxjs/toolkit";

import { Range } from "./types";

export interface WorkspaceState {
  selectedRangeKey: string | null;
  ranges: Record<string, Range>;
}

export interface WorkspaceStoreState {
  workspace: WorkspaceState;
}

export const initialState: WorkspaceState = {
  selectedRangeKey: null,
  ranges: {},
};

type AddRangeAction = PayloadAction<Range>;
type RemoveRangeAction = PayloadAction<string>;
type SelectRangeAction = PayloadAction<string | null>;

export const {
  actions: { addRange, removeRange, selectRange },
  reducer: workspaceReducer,
} = createSlice({
  name: "workspace",
  initialState,
  reducers: {
    addRange: (state, { payload }: AddRangeAction) => {
      state.ranges[payload.key] = payload;
    },
    removeRange: (state, { payload }: RemoveRangeAction) => {
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete state.ranges[payload];
    },
    selectRange: (state, { payload }: SelectRangeAction) => {
      state.selectedRangeKey = payload;
    },
  },
});
