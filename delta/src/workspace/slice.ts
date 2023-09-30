// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PayloadAction, createSlice } from "@reduxjs/toolkit";
import { type workspace } from "@synnaxlabs/client";

export interface SliceState {
  active: string | null;
  workspaces: Record<string, workspace.Workspace>;
}

export const SLICE_NAME = "workspace";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const initialState: SliceState = {
  active: null,
  workspaces: {},
};

type SetActivePayload = string | null;
export interface AddPayload {
  workspaces: workspace.Workspace[];
}

export interface RenamePayload {
  key: string;
  name: string;
}

type PA<P> = PayloadAction<P>;

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState,
  reducers: {
    setActive: (state, { payload }: PA<SetActivePayload>) => {
      state.active = payload;
    },
    add: (state, { payload: { workspaces } }: PA<AddPayload>) => {
      workspaces.forEach((workspace) => {
        state.workspaces[workspace.key] = workspace;
        state.active = workspace.key;
      });
    },
    rename: (state, { payload: { key, name } }: PA<RenamePayload>) => {
      state.workspaces[key].name = name;
    },
  },
});

export const { setActive, add, rename } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
