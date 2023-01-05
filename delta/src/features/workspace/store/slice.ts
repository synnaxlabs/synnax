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

import { Range } from "./types";

export interface WorkspaceState {
  activeRange: string | null;
  ranges: Record<string, Range>;
}

export interface WorkspaceStoreState {
  workspace: WorkspaceState;
}

export const WORKSPACE_SLICE_NAME = "workspace";

export const initialState: WorkspaceState = {
  activeRange: null,
  ranges: {},
};

type AddRangeAction = PayloadAction<Range>;
type RemoveRangeAction = PayloadAction<string>;
type SetActiveRangeAction = PayloadAction<string | null>;

export const {
  actions: { addRange, removeRange, setActiveRange },
  reducer: workspaceReducer,
} = createSlice({
  name: WORKSPACE_SLICE_NAME,
  initialState,
  reducers: {
    addRange: (state, { payload }: AddRangeAction) => {
      state.activeRange = payload.key;
      state.ranges[payload.key] = payload;
    },
    removeRange: (state, { payload }: RemoveRangeAction) => {
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete state.ranges[payload];
    },
    setActiveRange: (state, { payload }: SetActiveRangeAction) => {
      state.activeRange = payload;
    },
  },
});
