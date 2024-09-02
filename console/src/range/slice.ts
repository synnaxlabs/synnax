// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

import * as latest from "@/range/migrations";

export type SliceState = latest.SliceState;
export type DynamicRange = latest.DynamicRange;
export type StaticRange = latest.StaticRange;
export type Range = latest.Range;
export const migrateSlice = latest.migrateSlice;

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

export interface SetOffset {
  key: string;
  offset: number;
}

type SetActivePayload = string | null;

type PA<P> = PayloadAction<P>;

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: latest.ZERO_SLICE_STATE,
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
    setOffset: (state, { payload: { key, offset } }: PA<SetOffset>) => {
      const range = state.ranges[key];
      if (range == null || range.variant !== "static") return;
      range.offset = offset;
    },
  },
});
export const { add, remove, rename, setActive, setOffset } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
