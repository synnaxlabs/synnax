// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, PayloadAction } from "@reduxjs/toolkit";

/**
 * The name of the docs slice in a larger store.
 * NOTE: This must be the name of the slice in the store, or else all selectors will fail.
 */
export const DOCS_SLICE_NAME = "docs";

export interface DocsLocation {
  path: string;
  heading: string;
}

/** The state of the docs slice */
export interface DocsState {
  location: DocsLocation;
}

/**
 * The state of the docs slice within a larger store.
 * NOTE: This must be the name of the slice in the store, or else all selectors will fail.
 */
export interface DocsStoreState {
  [DOCS_SLICE_NAME]: DocsState;
}

const initialState: DocsState = {
  location: {
    path: "",
    heading: "",
  },
};

/** Payload for the setDocsPath action. */
export type SetDocsLocationPayload = DocsLocation;

export const {
  actions: {
    /**
     * Sets the path of the docs page.
     * @param path The path to set.
     * @returns The action to dispatch.
     * @category Actions
     */
    setDocsLocation,
  },
  reducer: docsReducer,
} = createSlice({
  name: DOCS_SLICE_NAME,
  initialState,
  reducers: {
    setDocsLocation: (state, action: PayloadAction<SetDocsLocationPayload>) => {
      state.location = action.payload;
    },
  },
});
