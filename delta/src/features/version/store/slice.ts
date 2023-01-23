// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice } from "@reduxjs/toolkit";
import type { PayloadAction } from "@reduxjs/toolkit";

export const VERSION_SLICE_NAME = "version";

export interface VersionState {
  version: string;
}

export interface VersionStoreState {
  [VERSION_SLICE_NAME]: VersionState;
}

const initialState: VersionState = {
  version: "0.0.0",
};

export type SetVersionAction = PayloadAction<string>;

export const {
  actions: { setVersion },
  reducer: versionReducer,
} = createSlice({
  name: VERSION_SLICE_NAME,
  initialState,
  reducers: {
    setVersion: (state, { payload: version }: SetVersionAction) => {
      state.version = version;
    },
  },
});
