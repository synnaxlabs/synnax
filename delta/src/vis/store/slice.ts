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
import { Deep, DeepPartial } from "@synnaxlabs/x";

import { VisMeta } from "@/vis/core";

export interface VisualizationState {
  warpMode: boolean;
  visualizations: Record<string, VisMeta>;
}

export interface VisualizationStoreState {
  visualization: VisualizationState;
}

export const initialState: VisualizationState = {
  warpMode: false,
  visualizations: {},
};

type SetVisAction = PayloadAction<VisMeta>;
type RemoveVisAction = PayloadAction<string>;
type UpdateVisAction = PayloadAction<
  Omit<DeepPartial<VisMeta>, "key"> & { key: string }
>;

export const VISUALIZATION_SLICE_NAME = "visualization";

export const {
  actions: { setVis, updateVis, removeVis },
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
    removeVis: (state, { payload }: RemoveVisAction) => {
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete state.visualizations[payload];
    },
  },
});
