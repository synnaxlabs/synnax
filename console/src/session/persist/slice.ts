// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice } from "@reduxjs/toolkit";

import { beginSwap, endSwap, hydrate } from "@/session/persist/state";
import { Select } from "@/session/select";

export const SLICE_NAME = "persist";

/**
 * Transient persistence status. Never persisted itself: the slice belongs to
 * no partition scope, so swaps and hydrates leave it untouched.
 */
export interface SliceState {
  /** Whether a partition swap is between flush and hydrate. */
  swapping: boolean;
}

export const ZERO_SLICE_STATE: SliceState = { swapping: false };

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

const slice = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {},
  extraReducers: (builder) => {
    builder
      .addCase(beginSwap, (state) => {
        state.swapping = true;
      })
      .addCase(endSwap, (state) => {
        state.swapping = false;
      })
      .addCase(hydrate, (state) => {
        state.swapping = false;
      });
  },
});

export const { reducer } = slice;

export const selectSwapping = (state: StoreState): boolean =>
  state[SLICE_NAME].swapping;

export const useSelectSwapping = (): boolean => Select.useMemo(selectSwapping, []);
