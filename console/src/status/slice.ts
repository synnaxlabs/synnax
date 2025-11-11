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
import { array } from "@synnaxlabs/x";

export const SLICE_NAME = "status";

export interface SliceState {
  favorites: status.Key[];
}

export const ZERO_SLICE_STATE: SliceState = { favorites: [] };

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

type AddFavoritesPayload = status.Key | status.Key[];

type RemoveFavoritesPayload = status.Key | status.Key[];

type ToggleFavoritePayload = status.Key;

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    addFavorites: (state, { payload: keys }: PayloadAction<AddFavoritesPayload>) => {
      const existingFavorites = new Set(state.favorites);
      for (const key of array.toArray(keys)) {
        if (existingFavorites.has(key)) continue;
        state.favorites.push(key);
        existingFavorites.add(key);
      }
    },
    removeFavorites: (
      state,
      { payload: keys }: PayloadAction<RemoveFavoritesPayload>,
    ) => {
      const favoritesToDelete = new Set(array.toArray(keys));
      if (favoritesToDelete.size === 0) return;
      state.favorites = state.favorites.filter((key) => !favoritesToDelete.has(key));
    },
    toggleFavorite: (state, { payload: key }: PayloadAction<ToggleFavoritePayload>) => {
      const existingIndex = state.favorites.indexOf(key);
      if (existingIndex !== -1) state.favorites.splice(existingIndex, 1);
      else state.favorites.push(key);
    },
  },
});

export const { addFavorites, removeFavorites, toggleFavorite } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const migrateSlice = (state: SliceState): SliceState => ({
  ...ZERO_SLICE_STATE,
  ...state,
});
