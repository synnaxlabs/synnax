// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { Color } from "@synnaxlabs/pluto";

export const SLICE_NAME = "color";

export interface SliceState {
  /**
   * The color picker's shared context: recent colors and frequency data reused
   * across pickers and windows.
   */
  context: Color.ContextState;
}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = { context: Color.ZERO_CONTEXT_STATE };

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setContext: (state, { payload }: PayloadAction<Color.ContextState>) => {
      state.context = payload;
    },
  },
});

export const { setContext } = actions;
export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
