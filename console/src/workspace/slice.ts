// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type workspace } from "@synnaxlabs/client";

import { type SliceState, type Workspace, ZERO_SLICE_STATE } from "@/workspace/types";

export const SLICE_NAME = "workspace";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface MaybeRenamePayload {
  key: workspace.Key;
  name: string;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setActive: (state, { payload }: PayloadAction<Workspace | null>) => {
      state.active = payload;
    },
    maybeRename: (
      state,
      { payload: { key, name } }: PayloadAction<MaybeRenamePayload>,
    ) => {
      if (state.active?.key !== key) return;
      state.active.name = name;
    },
  },
});

export const { setActive, maybeRename } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
