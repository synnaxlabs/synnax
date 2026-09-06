// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice } from "@reduxjs/toolkit";

export const SLICE_NAME = "link";

/**
 * Transient deep-link progress. A pending link belongs to the launch that received it.
 */
export interface SliceState {
  /** Whether a received link waits on a project selection before it can open. */
  awaitingProject: boolean;
}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = { awaitingProject: false };

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    beginProjectWait: (state) => {
      state.awaitingProject = true;
    },
    endProjectWait: (state) => {
      state.awaitingProject = false;
    },
  },
});

export const { beginProjectWait, endProjectWait } = actions;
export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
