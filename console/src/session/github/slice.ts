// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice } from "@reduxjs/toolkit";
import z from "zod";

export const SLICE_NAME = "github";

export const sliceStateZ = z
  .object({
    version: z.literal(0).default(0),
    /** True once the user starred Synnax on GitHub, or opened its page to star it. */
    starred: z.boolean().default(false),
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
    markStarred: (state) => {
      state.starred = true;
    },
  },
});

export const { markStarred } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
