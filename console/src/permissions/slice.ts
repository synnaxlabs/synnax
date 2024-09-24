// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type policy } from "@synnaxlabs/client";

import * as latest from "@/permissions/migrations";

export type SliceState = latest.SliceState;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;

export const SLICE_NAME = "permissions";
export const ALLOW_ALL = latest.ALLOW_ALL;

export type StoreState = {
  [SLICE_NAME]: SliceState;
};

export interface SetPayload {
  policies: policy.Policy[];
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    clear: (state) => {
      state.policies = [];
    },
    giveAllPermissions: (state) => {
      state.policies = ALLOW_ALL;
    },
    set: (state, { payload: { policies } }: PayloadAction<SetPayload>) => {
      state.policies = policies;
    },
  },
});

export const { clear, giveAllPermissions, set } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
