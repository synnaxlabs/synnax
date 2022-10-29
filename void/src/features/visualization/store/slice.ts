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

type CreateVisualizationAction = PayloadAction<Visualization>;

export const {
  actions: { createVisualization },
  reducer: visualizationReducer,
} = createSlice({
  name: "visualization",
  initialState,
  reducers: {
    createVisualization: (state, { payload }: PayloadAction<Visualization>) => {
      if (state.visualizations[payload.layoutKey]) {
        console.warn(
          `Visualization with key ${payload.layoutKey} already exists`
        );
        return;
      }
      state.visualizations[payload.layoutKey] = payload;
    },
  },
});
