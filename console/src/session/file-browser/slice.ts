// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import type { rack } from "@synnaxlabs/client";
import z from "zod";

export const SLICE_NAME = "fileBrowser";

const MAX_RECENT = 5;

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  /** The folders of the files picked on each rack, newest first. */
  recent: z.record(z.string(), z.string().array()).default({}),
});

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({});

interface AddRecentPayload {
  rack: rack.Key;
  folder: string;
}

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    addRecent: (
      state,
      { payload: { rack, folder } }: PayloadAction<AddRecentPayload>,
    ) => {
      const prev = state.recent[rack] ?? [];
      state.recent[rack] = [folder, ...prev.filter((f) => f !== folder)].slice(
        0,
        MAX_RECENT,
      );
    },
  },
});

export const { addRecent } = actions;
export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
