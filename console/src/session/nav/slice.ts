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

import { Window } from "@/session/window";

const baseStateZ = z.object({
  hover: z.boolean().default(false),
  size: z.number().optional(),
});
export const leftStateZ = baseStateZ
  .extend({ selected: z.string().optional() })
  .prefault({});
export interface LeftState extends z.infer<typeof leftStateZ> {}

export const bottomStateZ = baseStateZ
  .extend({ visible: z.boolean().default(false) })
  .prefault({});
export interface BottomState extends z.infer<typeof bottomStateZ> {}

export const windowStateZ = z
  .object({ left: leftStateZ, bottom: bottomStateZ })
  .prefault({});

export const ZERO_WINDOW_STATE = windowStateZ.parse({});

export interface WindowState extends z.output<typeof windowStateZ> {}

export const sliceStateZ = z.object({
  windows: z.record(z.string(), windowStateZ).default({}),
});

export interface SliceState extends z.output<typeof sliceStateZ> {}

export const SLICE_NAME = "nav";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({});

export interface LeftKeyPayload extends Window.OptionalKeyParams {
  key: string;
}

export interface ResizePayload extends Window.OptionalKeyParams {
  size: number;
}

const withKey = Window.createWithKeyHandler(windowStateZ);

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    selectLeft: withKey<LeftKeyPayload, SliceState>(
      ({ left }, { payload: { key } }) => {
        if (left.selected === key && !left.hover) left.selected = undefined;
        else {
          left.selected = key;
          left.hover = false;
        }
      },
    ),
    pinLeft: withKey<LeftKeyPayload, SliceState>(({ left }, { payload: { key } }) => {
      left.selected = key;
      left.hover = false;
    }),
    toggleLeft: withKey<LeftKeyPayload, SliceState>(
      ({ left }, { payload: { key } }) => {
        if (left.selected != null && !left.hover) {
          left.selected = left.selected === key ? undefined : key;
          return;
        }
        if (left.hover && key !== left.selected) {
          left.selected = key;
          return;
        }
        left.hover = !left.hover;
        left.selected = left.hover ? key : undefined;
      },
    ),
    startLeftHover: withKey<LeftKeyPayload, SliceState>(
      ({ left }, { payload: { key } }) => {
        if (left.selected != null && !left.hover) return;
        left.selected = key;
        left.hover = true;
      },
    ),
    stopLeftHover: withKey<Window.OptionalKeyParams, SliceState>(({ left }) => {
      if (!left.hover) return;
      left.hover = false;
      left.selected = undefined;
    }),
    resizeLeft: withKey<ResizePayload, SliceState>(
      ({ left }, { payload: { size } }) => {
        left.size = size;
      },
    ),
    collapseLeft: withKey<Window.OptionalKeyParams, SliceState>(({ left }) => {
      left.hover = false;
      left.selected = undefined;
    }),
    selectBottom: withKey<Window.OptionalKeyParams, SliceState>(({ bottom }) => {
      if (bottom.visible && !bottom.hover) bottom.visible = false;
      else {
        bottom.visible = true;
        bottom.hover = false;
      }
    }),
    showBottom: withKey<Window.OptionalKeyParams, SliceState>(({ bottom }) => {
      bottom.visible = true;
      bottom.hover = false;
    }),
    toggleBottom: withKey<Window.OptionalKeyParams, SliceState>(({ bottom }) => {
      if (bottom.visible && !bottom.hover) bottom.visible = false;
      else if (bottom.visible && bottom.hover) bottom.hover = false;
      else {
        bottom.visible = true;
        bottom.hover = true;
      }
    }),
    startBottomHover: withKey<Window.OptionalKeyParams, SliceState>(({ bottom }) => {
      if (bottom.visible && !bottom.hover) return;
      bottom.visible = true;
      bottom.hover = true;
    }),
    stopBottomHover: withKey<Window.OptionalKeyParams, SliceState>(({ bottom }) => {
      if (!bottom.hover) return;
      bottom.hover = false;
      bottom.visible = false;
    }),
    resizeBottom: withKey<ResizePayload, SliceState>(
      ({ bottom }, { payload: { size } }) => {
        bottom.size = size;
      },
    ),
    collapseBottom: withKey<Window.OptionalKeyParams, SliceState>(({ bottom }) => {
      bottom.hover = false;
      bottom.visible = false;
    }),
    hideAll: withKey<Window.OptionalKeyParams, SliceState>(({ left, bottom }) => {
      left.selected = undefined;
      left.hover = false;
      bottom.visible = false;
      bottom.hover = false;
    }),
  },
});

export const {
  selectLeft,
  pinLeft,
  toggleLeft,
  startLeftHover,
  stopLeftHover,
  resizeLeft,
  collapseLeft,
  selectBottom,
  showBottom,
  toggleBottom,
  startBottomHover,
  stopBottomHover,
  resizeBottom,
  collapseBottom,
  hideAll,
} = actions;

export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];

export const MIDDLEWARE = [
  Window.createInjectKeyMiddleware([
    selectLeft,
    pinLeft,
    toggleLeft,
    startLeftHover,
    stopLeftHover,
    resizeLeft,
    collapseLeft,
    selectBottom,
    showBottom,
    toggleBottom,
    startBottomHover,
    stopBottomHover,
    resizeBottom,
    collapseBottom,
    hideAll,
  ]),
];
