// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { project } from "@synnaxlabs/client";
import z from "zod";

const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  selected: project.keyZ.optional(),
});

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "project";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    select: (state, { payload: key }: PayloadAction<project.Key>) => {
      state.selected = key;
    },
    clearSelected: (state) => {
      state.selected = undefined;
    },
  },
});

export const { select, clearSelected } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
