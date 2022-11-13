import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { Range } from "./types";

export interface WorkspaceState {
	ranges: Range[];
}

export interface WorkspaceStoreState {
	workspace: WorkspaceState;
}

export const initialState: WorkspaceState = {
	ranges: [],
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
			console.log(payload);
			state.ranges.push(payload);
		},
		removeRange: (state, { payload }: RemoveRangeAction) => {
			state.ranges = state.ranges.filter((range) => range.key !== payload);
		},
	},
});
