// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { Haul } from "@synnaxlabs/pluto";

export const SLICE_NAME = "haul";

export interface SliceState {
  /**
   * The drag-and-drop payload currently being hauled, mirrored into redux so drags
   * synchronize across windows.
   */
  state: Haul.DraggingState;
}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = { state: Haul.ZERO_DRAGGING_STATE };

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setHauled: (state, { payload }: PayloadAction<Haul.DraggingState>) => {
      state.state = payload;
    },
  },
});

export const { setHauled } = actions;
export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];

/** Drag state is transient; a persisted drag would resume a phantom haul on relaunch. */
export const PERSIST_EXCLUDE = <S extends StoreState>(state: S): S => ({
  ...state,
  [SLICE_NAME]: ZERO_SLICE_STATE,
});
