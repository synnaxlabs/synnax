// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type workspace } from "@synnaxlabs/client";
import { toArray } from "@synnaxlabs/x";

import * as latest from "@/workspace/types";

export type SliceState = latest.SliceState;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;

export const SLICE_NAME = "workspace";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

type SetActivePayload = string | null;

export type AddPayload = workspace.Workspace | workspace.Workspace[];

export interface RemovePayload {
  keys: string[];
}

export interface RenamePayload {
  key: string;
  name: string;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setActive: (state, { payload }: PayloadAction<SetActivePayload>) => {
      state.active = payload;
    },
    add: (state, { payload: workspaces }: PayloadAction<AddPayload>) => {
      toArray(workspaces).forEach((workspace) => {
        state.workspaces[workspace.key] = workspace;
        state.active = workspace.key;
      });
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      keys.forEach((key) => {
        if (state.active === key) state.active = null;
        delete state.workspaces[key];
      });
    },
    rename: (state, { payload: { key, name } }: PayloadAction<RenamePayload>) => {
      state.workspaces[key].name = name;
    },
  },
});

export const { setActive, add, remove, rename } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
