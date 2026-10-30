// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

export const SLICE_NAME = "analytics";

/** Transient analytics state that every window of a launch shares. */
export interface SliceState {
  /** The PostHog session that every window reports in, once the main window has one. */
  sessionID?: string;
}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = {};

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setSessionID: (state, { payload }: PayloadAction<string>) => {
      state.sessionID = payload;
    },
  },
});

export const { setSessionID } = actions;
export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
