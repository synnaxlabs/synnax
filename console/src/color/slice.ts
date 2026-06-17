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
import { type z } from "zod";

export const SLICE_NAME = "color";

export const sliceStateZ = Color.contextStateZ;

export interface SliceState extends z.output<typeof sliceStateZ> {}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = Color.ZERO_CONTEXT_STATE;

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setContext: (_, { payload }: PayloadAction<Color.ContextState>) => payload,
  },
});

export const { setContext } = actions;

export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
