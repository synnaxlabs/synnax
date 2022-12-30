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
