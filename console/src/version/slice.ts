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
import { migrate } from "@synnaxlabs/x";

export const SLICE_NAME = "version";

export interface SliceState {
  version: string;
}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

const initialState: SliceState = {
  version: "0.0.0",
};

export type SetVersionAction = PayloadAction<string>;

export const MIGRATIONS: migrate.Migrations = {};

export const migrateSlice = migrate.migrator<SliceState, SliceState>(MIGRATIONS);

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState,
  reducers: {
    set: (state, { payload: version }: SetVersionAction) => {
      state.version = version;
    },
  },
});

export const { set } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
