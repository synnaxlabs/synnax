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

export const LEFT_ITEMS = [
  "channel",
  "range",
  "device",
  "task",
  "user",
  "project",
  "arc",
  "status",
] as const;

export const leftItemZ = z.enum(LEFT_ITEMS);
export type LeftItem = z.output<typeof leftItemZ>;

const baseDrawerStateZ = z.object({
  hover: z.boolean().default(false),
  size: z.number().optional(),
});

export const sliceStateZ = z
  .object({
    left: baseDrawerStateZ.extend({ selected: leftItemZ.optional() }).prefault({}),
    bottom: baseDrawerStateZ
      .extend({ visible: z.boolean().default(false) })
      .prefault({}),
  })
  .prefault({});

export interface SliceState extends z.output<typeof sliceStateZ> {}

export const SLICE_NAME = "nav";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({});

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    selectLeft: (state, { payload }: PayloadAction<LeftItem>) => {
      const { left } = state;
      if (left.selected === payload && !left.hover) left.selected = undefined;
      else {
        left.selected = payload;
        left.hover = false;
      }
    },
    pinLeft: (state, { payload }: PayloadAction<LeftItem>) => {
      state.left.selected = payload;
      state.left.hover = false;
    },
    toggleLeft: (state, { payload }: PayloadAction<LeftItem>) => {
      const { left } = state;
      if (left.selected != null && !left.hover) {
        left.selected = left.selected === payload ? undefined : payload;
        return;
      }
      if (left.hover && payload !== left.selected) {
        left.selected = payload;
        return;
      }
      left.hover = !left.hover;
      left.selected = left.hover ? payload : undefined;
    },
    startLeftHover: (state, { payload }: PayloadAction<LeftItem>) => {
      const { left } = state;
      if (left.selected != null && !left.hover) return;
      left.selected = payload;
      left.hover = true;
    },
    stopLeftHover: (state) => {
      const { left } = state;
      if (!left.hover) return;
      left.hover = false;
      left.selected = undefined;
    },
    resizeLeft: (state, { payload }: PayloadAction<number>) => {
      state.left.size = payload;
    },
    selectBottom: (state) => {
      const { bottom } = state;
      if (bottom.visible && !bottom.hover) bottom.visible = false;
      else {
        bottom.visible = true;
        bottom.hover = false;
      }
    },
    setBottomVisible: (state, { payload }: PayloadAction<boolean>) => {
      state.bottom.visible = payload;
      state.bottom.hover = false;
    },
    toggleBottom: (state) => {
      const { bottom } = state;
      if (bottom.visible && !bottom.hover) bottom.visible = false;
      else if (bottom.visible && bottom.hover) bottom.hover = false;
      else {
        bottom.visible = true;
        bottom.hover = true;
      }
    },
    startBottomHover: (state) => {
      const { bottom } = state;
      if (bottom.visible && !bottom.hover) return;
      bottom.visible = true;
      bottom.hover = true;
    },
    stopBottomHover: (state) => {
      const { bottom } = state;
      if (!bottom.hover) return;
      bottom.hover = false;
      bottom.visible = false;
    },
    resizeBottom: (state, { payload }: PayloadAction<number>) => {
      state.bottom.size = payload;
    },
    hideAll: (state) => {
      state.left.selected = undefined;
      state.left.hover = false;
      state.bottom.visible = false;
      state.bottom.hover = false;
    },
  },
});

export const {
  selectLeft,
  pinLeft,
  toggleLeft,
  startLeftHover,
  stopLeftHover,
  resizeLeft,
  selectBottom,
  setBottomVisible,
  toggleBottom,
  startBottomHover,
  stopBottomHover,
  resizeBottom,
  hideAll,
} = actions;

export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
