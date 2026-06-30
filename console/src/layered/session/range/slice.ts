// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type ranger } from "@synnaxlabs/client";
import { type NumericTimeRange, numericTimeRangeZ, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

export const baseZ = z.object({
  key: z.string(),
  name: z.string(),
  persisted: z.boolean(),
});

export const staticZ = baseZ.extend({
  variant: z.literal("static"),
  timeRange: numericTimeRangeZ,
});

export interface Static extends z.infer<typeof staticZ> {}

export const dynamicZ = baseZ.extend({
  variant: z.literal("dynamic"),
  span: z.number(),
});

export interface Dynamic extends z.infer<typeof dynamicZ> {}

export const rangeZ = z.union([staticZ, dynamicZ]);

export type Range = z.infer<typeof rangeZ>;

export const RECENT_KEY = "recent";

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  selected: z.string().optional(),
  ranges: z.array(rangeZ).default([
    {
      key: RECENT_KEY,
      variant: "dynamic",
      name: "Rolling 30s",
      span: Number(TimeSpan.seconds(30)),
      persisted: false,
    },
    {
      key: "rolling1m",
      variant: "dynamic",
      name: "Rolling 1m",
      span: Number(TimeSpan.minutes(1)),
      persisted: false,
    },
    {
      key: "rolling5m",
      variant: "dynamic",
      name: "Rolling 5m",
      span: Number(TimeSpan.minutes(5)),
      persisted: false,
    },
    {
      key: "rolling15m",
      variant: "dynamic",
      name: "Rolling 15m",
      span: Number(TimeSpan.minutes(15)),
      persisted: false,
    },
    {
      key: "rolling30m",
      variant: "dynamic",
      name: "Rolling 30m",
      span: Number(TimeSpan.minutes(30)),
      persisted: false,
    },
  ]),
});

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export type SliceState = z.infer<typeof sliceStateZ>;

export const SLICE_NAME = "range";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type AddPayload = Range & { switchActive?: boolean };

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

type SelectPayload = string;

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    add: (
      state,
      { payload: { switchActive = true, ...range } }: PayloadAction<AddPayload>,
    ) => {
      if (switchActive === true) state.selected = range.key;
      state.ranges = [...state.ranges.filter((r) => r.key !== range.key), range];
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      if (state.selected != null && keys.includes(state.selected))
        state.selected = undefined;
      state.ranges = state.ranges.filter(({ key }) => !keys.includes(key));
    },
    select: (state, { payload }: PayloadAction<SelectPayload>) => {
      state.selected = payload;
    },
    rename: (state, { payload: { key, name } }: PayloadAction<RenamePayload>) => {
      const r = state.ranges.find((r) => r.key === key);
      if (r == null) return;
      r.name = name;
    },
    updateRemote: (
      state,
      { payload: { key, name, timeRange } }: PayloadAction<UpdateIfExistsPayload>,
    ) => {
      const r = state.ranges.find((r) => r.key === key);
      if (r == null || r.variant == "dynamic") return;
      r.name = name;
      r.timeRange = timeRange;
    },
  },
});
export const { add, remove, rename, select, updateRemote } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
