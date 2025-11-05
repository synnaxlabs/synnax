// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type status } from "@synnaxlabs/client";

export const SLICE_NAME = "status";

export interface SliceState {
  favorites: status.Key[];
}

export const ZERO_SLICE_STATE: SliceState = {
  favorites: [],
};

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

interface ToggleFavoritePayload {
  key: status.Key;
}

type PA<P> = PayloadAction<P>;

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    toggleFavorite: (state, { payload: { key } }: PA<ToggleFavoritePayload>) => {
      const index = state.favorites.indexOf(key);
      if (index === -1) state.favorites.push(key);
      else state.favorites.splice(index, 1);
    },
    removeFavorite: (state, { payload: { key } }: PA<ToggleFavoritePayload>) => {
      const index = state.favorites.indexOf(key);
      if (index !== -1) state.favorites.splice(index, 1);
    },
  },
});

export const { toggleFavorite, removeFavorite } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const migrateSlice = (state: SliceState): SliceState => ({
  ...ZERO_SLICE_STATE,
  ...state,
});
