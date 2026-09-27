// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { array, numericTimeRangeZ } from "@synnaxlabs/x";
import { z } from "zod";

/**
 * A range the Core holds. Only the key is kept: the name and time range live on the
 * Core, so a copy here could only go stale.
 */
export const persistedStateZ = z.object({
  variant: z.literal("persisted"),
  key: z.string(),
});

export interface PersistedState extends z.infer<typeof persistedStateZ> {}

/** A fixed window the session owns, saved to the Core only on request. */
export const staticStateZ = z.object({
  variant: z.literal("static"),
  key: z.string(),
  name: z.string(),
  timeRange: numericTimeRangeZ,
});

export interface StaticState extends z.infer<typeof staticStateZ> {}

export const stateZ = z.discriminatedUnion("variant", [persistedStateZ, staticStateZ]);

export type State = z.infer<typeof stateZ>;

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  selected: z.string().optional(),
  /** The ranges the session added. */
  ranges: z.array(stateZ).default([]),
});

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export type SliceState = z.infer<typeof sliceStateZ>;

export const SLICE_NAME = "range";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type AddPayload = State | State[];

interface RemovePayload {
  keys: string[];
}

interface RestorePayload {
  ranges: { index: number; range: State }[];
  selected?: string;
}

interface RenamePayload {
  key: string;
  name: string;
}

type SelectPayload = string;

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    add: (state, { payload: range }: PayloadAction<AddPayload>) => {
      const ranges = array.toArray(range);
      const keys = ranges.map(({ key }) => key);
      state.ranges = [...state.ranges.filter((r) => !keys.includes(r.key)), ...ranges];
      if (ranges.length > 0) state.selected = ranges[ranges.length - 1].key;
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      if (state.selected != null && keys.includes(state.selected))
        state.selected = undefined;
      state.ranges = state.ranges.filter(({ key }) => !keys.includes(key));
    },
    restore: (
      state,
      { payload: { ranges, selected } }: PayloadAction<RestorePayload>,
    ) => {
      ranges
        .filter(({ range }) => !state.ranges.some(({ key }) => key === range.key))
        // Ascending order keeps each splice from shifting the ones behind it.
        .sort((a, b) => a.index - b.index)
        .forEach(({ index, range }) => state.ranges.splice(index, 0, range));
      if (selected != null) state.selected = selected;
    },
    select: (state, { payload }: PayloadAction<SelectPayload>) => {
      state.selected = payload;
    },
    clearSelected: (state) => {
      state.selected = undefined;
    },
    // A persisted range is renamed on the Core, which is where its name lives.
    rename: (state, { payload: { key, name } }: PayloadAction<RenamePayload>) => {
      const r = state.ranges.find((r) => r.key === key);
      if (r == null || r.variant === "persisted") return;
      r.name = name;
    },
  },
});
export const { add, clearSelected, remove, rename, restore, select } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
