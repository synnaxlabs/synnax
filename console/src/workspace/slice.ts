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
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

export interface SliceState extends migrate.Migratable {
  active: string | null;
  workspaces: Record<string, workspace.Workspace>;
}

export const SLICE_NAME = "workspace";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.0",
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

export const MIGRATIONS: migrate.Migrations = {};

export const migrateSlice = migrate.migrator({
  name: "workspace.slice",
  migrations: MIGRATIONS,
  target: z.any(),
  def: ZERO_SLICE_STATE,
});

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setActive: (state, { payload }: PayloadAction<SetActivePayload>) => {
      state.active = payload;
    },
    add: (state, { payload: { workspaces } }: PayloadAction<AddPayload>) => {
      workspaces.forEach((workspace) => {
        state.workspaces[workspace.key] = workspace;
        state.active = workspace.key;
      });
    },
    rename: (state, { payload: { key, name } }: PayloadAction<RenamePayload>) => {
      state.workspaces[key].name = name;
    },
  },
});

export const { setActive, add, rename } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
