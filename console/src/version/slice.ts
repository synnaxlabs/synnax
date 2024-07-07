// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { PayloadAction } from "@reduxjs/toolkit";
import { createSlice } from "@reduxjs/toolkit";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

export const SLICE_NAME = "version";

export interface SliceState {
  version: string;
}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.0",
};

export type SetVersionAction = PayloadAction<string>;

export const MIGRATIONS: migrate.Migrations = {};

export const migrateSlice = migrate.migrator({
  name: "version.slice",
  migrations: MIGRATIONS,
  def: ZERO_SLICE_STATE,
});

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    set: (state, { payload: version }: SetVersionAction) => {
      state.version = version;
    },
  },
});

export const { set } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
