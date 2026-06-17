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
import { type z } from "zod";

export const SLICE_NAME = "hauling";

export const sliceStateZ = Haul.draggingStateZ;

export interface SliceState extends z.output<typeof sliceStateZ> {}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = Haul.ZERO_DRAGGING_STATE;

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setHauled: (_, { payload }: PayloadAction<Haul.DraggingState>) => payload,
  },
});

export const { setHauled } = actions;

export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
