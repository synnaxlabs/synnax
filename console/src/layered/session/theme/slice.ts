// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import z from "zod";

export const SLICE_NAME = "theme";

const sliceStateZ = z
  .object({
    selected: z.string().default("synnaxLight"),
  })
  .prefault({});

export interface SliceState extends z.output<typeof sliceStateZ> {}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({});

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    select: (state, action: PayloadAction<string>) => {
      state.selected = action.payload;
    },
    toggle: (state) => {
      state.selected = state.selected === "synnaxLight" ? "synnaxDark" : "synnaxLight";
    },
  },
});

export const { select, toggle } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
