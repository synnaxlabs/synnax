// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { migrate } from "@synnaxlabs/x";

/**
 * The name of the docs slice in a larger store.
 * NOTE: This must be the name of the slice in the store, or else all selectors will fail.
 */
export const SLICE_NAME = "docs";

export interface Location {
  path: string;
  heading: string;
}

/** The state of the docs slice */
export interface SliceState extends migrate.Migratable {
  location: Location;
}

/**
 * The state of the docs slice within a larger store.
 * NOTE: This must be the name of the slice in the store, or else all selectors will fail.
 */
export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.0",
  location: {
    path: "",
    heading: "",
  },
};

/** Payload for the setDocsPath action. */
export type SetLocationPayload = Location;

export const MIGRATIONS: migrate.Migrations = {};

export const migrateSlice = migrate.migrator<SliceState, SliceState>({
  name: "docs.slice",
  migrations: MIGRATIONS,
  def: ZERO_SLICE_STATE,
});

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setDocsLocation: (state, action: PayloadAction<SetLocationPayload>) => {
      state.location = action.payload;
    },
  },
});

export const {
  /**
   * Sets the path of the docs page.
   * @param path The path to set.
   * @returns The action to dispatch.
   * @category Actions
   */
  setDocsLocation,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
