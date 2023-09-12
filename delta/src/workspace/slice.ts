// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice } from "@reduxjs/toolkit";
import type { PayloadAction } from "@reduxjs/toolkit";
import { TimeSpan } from "@synnaxlabs/x";

import { type Range } from "@/workspace/range";

export interface SliceState {
  activeRange: string | null;
  ranges: Record<string, Range>;
}

export interface StoreState {
  workspace: SliceState;
}

export const SLICE_NAME = "workspace";

export const initialState: SliceState = {
  activeRange: null,
  ranges: {
    recent: {
      key: "recent",
      variant: "dynamic",
      name: "Recent",
      span: Number(TimeSpan.minutes(60)),
    },
    hour: {
      key: "hour",
      variant: "dynamic",
      name: "Recent",
      span: Number(TimeSpan.minutes(60)),
    },
  },
};

type AddRangePayload = Range;
type RemoveRangePayload = string;
type SetActiveRangePayload = string | null;

type PA<P> = PayloadAction<P>;

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState,
  reducers: {
    addRange: (state, { payload }: PA<AddRangePayload>) => {
      state.activeRange = payload.key;
      state.ranges[payload.key] = payload;
    },
    removeRange: (state, { payload }: PA<RemoveRangePayload>) => {
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete state.ranges[payload];
    },
    setActiveRange: (state, { payload }: PA<SetActiveRangePayload>) => {
      state.activeRange = payload;
    },
  },
});
export const { addRange, removeRange, setActiveRange } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
