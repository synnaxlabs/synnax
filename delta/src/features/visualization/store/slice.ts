import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { Visualization } from "../types";

export type VisualizationState = {
  visualizations: Record<string, Visualization>;
};

export interface VisualizationStoreState {
  visualization: VisualizationState;
}

export const initialState: VisualizationState = {
  visualizations: {},
};

type SetVisualizationAction = PayloadAction<Visualization>;

export const {
  actions: { setVisualization },
  reducer: visualizationReducer,
} = createSlice({
  name: "visualization",
  initialState,
  reducers: {
    setVisualization: (state, { payload }: SetVisualizationAction) => {
      state.visualizations[payload.layoutKey] = payload;
    },
  },
});
