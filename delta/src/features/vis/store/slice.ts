// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice } from "@reduxjs/toolkit";
import type { PayloadAction } from "@reduxjs/toolkit";
import { Deep } from "@synnaxlabs/x";
import { DeepPartial } from "react-hook-form";

import { Vis } from "../types";

export interface VisualizationState {
  warpMode: boolean;
  visualizations: Record<string, Vis>;
}

export interface VisualizationStoreState {
  visualization: VisualizationState;
}

export const initialState: VisualizationState = {
  warpMode: false,
  visualizations: {},
};

type SetVisAction = PayloadAction<Vis>;
type UpdateVisAction = PayloadAction<Omit<DeepPartial<Vis>, "key"> & { key: string }>;
type SetWarpModeAction = PayloadAction<boolean | undefined>;

export const VISUALIZATION_SLICE_NAME = "visualization";

export const {
  actions: { setVis, setWarpMode, updateVis },
  reducer: visualizationReducer,
} = createSlice({
  name: VISUALIZATION_SLICE_NAME,
  initialState,
  reducers: {
    setVis: (state, { payload }: SetVisAction) => {
      state.visualizations[payload.key] = payload;
    },
    updateVis: (state, { payload }: UpdateVisAction) => {
      const vis = state.visualizations[payload.key];
      const res = Deep.merge(vis, payload);
      state.visualizations[payload.key] = res;
    },
    setWarpMode: (state, { payload }: SetWarpModeAction) => {
      state.warpMode = payload ?? !state.warpMode;
    },
  },
});
