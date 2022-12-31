// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice } from "@reduxjs/toolkit";
import type { PayloadAction } from "@reduxjs/toolkit";

import { Visualization } from "../types";

export interface VisualizationState {
  visualizations: Record<string, Visualization>;
}

export interface VisualizationStoreState {
  visualization: VisualizationState;
}

export const initialState: VisualizationState = {
  visualizations: {},
};

type SetVisualizationAction = PayloadAction<Visualization>;

export const VISUALIZATION_SLICE_NAME = "visualization";

export const {
  actions: { setVisualization },
  reducer: visualizationReducer,
} = createSlice({
  name: VISUALIZATION_SLICE_NAME,
  initialState,
  reducers: {
    setVisualization: (state, { payload }: SetVisualizationAction) => {
      state.visualizations[payload.layoutKey] = payload;
    },
  },
});
