// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type user } from "@synnaxlabs/client";

import * as latest from "@/user/types";

export type SliceState = latest.SliceState;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;

export const SLICE_NAME = "user";

export type StoreState = {
  [SLICE_NAME]: SliceState;
};

export interface SetPayload {
  user: user.User;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    set: (state, { payload: { user } }: PayloadAction<SetPayload>) => {
      state.user = user;
    },
    clear: (state) => {
      state.user = ZERO_SLICE_STATE.user;
    },
  },
});

export const { set, clear } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
