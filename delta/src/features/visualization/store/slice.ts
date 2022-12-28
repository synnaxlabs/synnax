import { createSlice } from "@reduxjs/toolkit";
import type { PayloadAction } from "@reduxjs/toolkit";

import { Visualization } from "../types";

import { mergeDeep } from "@/util/merge";

export interface VisualizationState {
  warpMode: boolean;
  visualizations: Record<string, Visualization>;
}

export interface VisualizationStoreState {
  visualization: VisualizationState;
}

export const initialState: VisualizationState = {
  warpMode: true,
  visualizations: {},
};

type SetVisualizationAction = PayloadAction<Visualization>;
type UpdateVisualizationAction = PayloadAction<
  Omit<Partial<Visualization>, "key"> & { key: string }
>;
type SetWarpModeAction = PayloadAction<boolean | undefined>;

export const VISUALIZATION_SLICE_NAME = "visualization";

export const {
  actions: { setVisualization, setWarpMode, updateVisualization },
  reducer: visualizationReducer,
} = createSlice({
  name: VISUALIZATION_SLICE_NAME,
  initialState,
  reducers: {
    setVisualization: (state, { payload }: SetVisualizationAction) => {
      state.visualizations[payload.key] = payload;
    },
    updateVisualization: (state, { payload }: UpdateVisualizationAction) => {
      const vis = state.visualizations[payload.key];
      if (vis == null) throw new Error(`visualization ${payload.key} does not exist`);
      const res = mergeDeep(vis, payload);
      state.visualizations[payload.key] = res;
    },
    setWarpMode: (state, { payload }: SetWarpModeAction) => {
      state.warpMode = payload ?? !state.warpMode;
    },
  },
});
