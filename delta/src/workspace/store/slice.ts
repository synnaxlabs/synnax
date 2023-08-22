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

import { Range } from "./types";

export interface WorkspaceSliceState {
  activeRange: string | null;
  ranges: Record<string, Range>;
}

export interface WorkspaceStoreState {
  workspace: WorkspaceSliceState;
}

export const WORKSPACE_SLICE_NAME = "workspace";

export const initialState: WorkspaceSliceState = {
  activeRange: null,
  ranges: {
    dog: {
      key: "dog",
      name: "Dog",
      variant: "dynamic",
      span: Number(TimeSpan.seconds(10)),
    },
  },
};

type AddRangePayload = Range;
type RemoveRangePayload = string;
type SetActiveRangePayload = string | null;

type PA<P> = PayloadAction<P>;

export const { actions, reducer: workspaceReducer } = createSlice({
  name: WORKSPACE_SLICE_NAME,
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

export type WorkspaceAction = ReturnType<(typeof actions)[keyof typeof actions]>;
export type WorkspacePayload = WorkspaceAction["payload"];
