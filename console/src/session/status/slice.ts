// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { status } from "@synnaxlabs/client";
import { array, uuid } from "@synnaxlabs/x";
import z from "zod";

export const SLICE_NAME = "status";

// Core v0.61 moved the status of each of these owners from the key "<type>:<key>" to
// the UUIDv5 of that key in this namespace.
const MOVED_OWNERS = new Set(["rack", "task", "device", "channel"]);
const MOVED_NAMESPACE = "4a3c5e0e-6d0b-5f43-9a51-3c7e1f2b8d64";

const migrateKey = (key: status.Key): status.Key => {
  const sep = key.indexOf(":");
  if (sep <= 0 || !MOVED_OWNERS.has(key.slice(0, sep))) return key;
  return uuid.fromName(key, MOVED_NAMESPACE);
};

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  favorites: status.keyZ
    .array()
    .default([])
    .transform((keys) => keys.map(migrateKey)),
});

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export interface SliceState extends z.infer<typeof sliceStateZ> {}

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
