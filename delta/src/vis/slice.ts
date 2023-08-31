// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice } from "@reduxjs/toolkit";
import { Viewport } from "@synnaxlabs/pluto";

export interface SliceState {
  viewportMode: Viewport.Mode;
}

export const SLICE_NAME = "vis";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

const ZERO_SLICE_STATE: SliceState = {
  viewportMode: "pan",
};

export interface SetVisViewportModePayload {
  mode: Viewport.Mode;
}

export const {
  actions: { setViewportMode },
  reducer,
} = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setViewportMode: (state, action) => {
      state.viewportMode = action.payload.mode;
    },
  },
});
