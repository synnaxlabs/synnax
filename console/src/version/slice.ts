// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

import * as latest from "@/version/types";

export type SliceState = latest.SliceState;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;

export const SLICE_NAME = "version";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}
export type SetVersionAction = PayloadAction<string>;

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    set: (state, { payload: version }: SetVersionAction) => {
      if (state.consoleVersion === version) return;
      state.consoleVersion = version;
      state.updateNotificationsSilenced = false;
    },
    silenceUpdateNotifications: (state) => {
      state.updateNotificationsSilenced = true;
    },
  },
});

export const { set, silenceUpdateNotifications } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
