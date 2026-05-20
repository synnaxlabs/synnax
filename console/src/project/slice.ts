// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type project } from "@synnaxlabs/client";

import { type Project, type SliceState, ZERO_SLICE_STATE } from "@/project/types";

export const SLICE_NAME = "project";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface MaybeRenamePayload {
  key: project.Key;
  name: string;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setActive: (state, { payload }: PayloadAction<Project | null>) => {
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
