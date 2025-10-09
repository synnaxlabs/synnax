// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type ranger } from "@synnaxlabs/client";
import { type NumericTimeRange } from "@synnaxlabs/x";

import * as latest from "@/range/types";

export type SliceState = latest.SliceState;
export type DynamicRange = latest.DynamicRange;
export type StaticRange = latest.StaticRange;
export type Range = latest.Range;
export const migrateSlice = latest.migrateSlice;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;

export const SLICE_NAME = "range";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

interface AddPayload {
  ranges: Range[];
  switchActive?: boolean;
}

interface RemovePayload {
  keys: string[];
}

interface RenamePayload {
  key: string;
  name: string;
}

interface UpdateIfExistsPayload extends Omit<ranger.Payload, "timeRange"> {
  timeRange: NumericTimeRange;
}

type SetActivePayload = string | null;

type PA<P> = PayloadAction<P>;

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    add: (state, { payload: { ranges, switchActive = true } }: PA<AddPayload>) => {
      ranges.forEach((range) => {
        if (switchActive === true) state.activeRange = range.key;
        state.ranges[range.key] = range;
      });
    },
    remove: (state, { payload: { keys } }: PA<RemovePayload>) => {
      if (state.activeRange != null && keys.includes(state.activeRange))
        state.activeRange = null;
      keys.forEach((k) => delete state.ranges[k]);
    },
    setActive: (state, { payload }: PA<SetActivePayload>) => {
      state.activeRange = payload;
    },
    rename: (state, { payload: { key, name } }: PA<RenamePayload>) => {
      const range = state.ranges[key];
      if (range == null) return;
      range.name = name;
    },
    updateIfExists: (
      state,
      { payload: { key, name, timeRange } }: PA<UpdateIfExistsPayload>,
    ) => {
      const range = state.ranges[key];
      if (range == null || range.variant === "dynamic") return;
      range.name = name;
      range.timeRange = timeRange;
    },
  },
});
export const { add, remove, rename, setActive, updateIfExists } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
