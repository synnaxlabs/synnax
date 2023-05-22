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
import { LineVis } from "@/vis/line/LinePlot/core";

export interface VisState {
  warpMode: boolean;
  visualizations: Record<string, VisMeta>;
}

export interface VisStoreState {
  visualization: VisState;
}

export const initialState: VisState = {
  warpMode: false,
  visualizations: {},
};

type SetVisPayload = VisMeta;
type RemoveVisPayload = string;
type UpdateVisPayload = Omit<DeepPartial<VisMeta>, "key"> & { key: string };

export const VIS_SLICE_NAME = "visualization";

export const { actions, reducer: visReducer } = createSlice({
  name: VIS_SLICE_NAME,
  initialState,
  reducers: {
    setVis: (state, { payload }: PayloadAction<SetVisPayload>) => {
      state.visualizations[payload.key] = payload;
    },
    updateVis: (state, { payload }: PayloadAction<UpdateVisPayload>) => {
      const vis = state.visualizations[payload.key];
      const res = Deep.merge(vis, payload);
      state.visualizations[payload.key] = res;
    },
    removeVis: (state, { payload }: PayloadAction<RemoveVisPayload>) => {
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete state.visualizations[payload];
    },
    purgeRanges: (state, { payload }: PayloadAction<RemoveVisPayload>) => {
      // iterate through all vis and remove any that have a range key that matches the payload
      Object.keys(state.visualizations).forEach((key) => {
        const vis = state.visualizations[key];
        if (vis.variant === "line") {
          const vis_ = vis as LineVis;
          vis_.ranges.x1 = vis_.ranges.x1.filter((r) => r !== payload);
          vis_.ranges.x2 = vis_.ranges.x2.filter((r) => r !== payload);
          state.visualizations[key] = vis_;
        }
      });
    },
  },
});

export const { setVis, updateVis, removeVis, purgeRanges } = actions;

export type VisAction = ReturnType<typeof actions[keyof typeof actions]>;
export type VisPayload = VisAction["payload"];
