// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

import * as latest from "@/log/types";

export type State = latest.State;
export type SliceState = latest.SliceState;
export type ToolbarTab = latest.ToolbarTab;
export type ToolbarState = latest.ToolbarState;
export type PendingUpload = latest.PendingUpload;
export const ZERO_TOOLBAR_STATE = latest.ZERO_TOOLBAR_STATE;
export const stateZ = latest.stateZ;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const ZERO_STATE = latest.ZERO_STATE;
export const migrateSlice = latest.migrateSlice;

export const SLICE_NAME = "log";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface CreatePayload {
  key: string;
}

export interface SetActiveToolbarTabPayload {
  key: string;
  tab: ToolbarTab;
}

export interface ClearPendingUploadPayload {
  key: string;
}

export interface RemovePayload {
  keys: string[];
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: latest.ZERO_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      if (state.logs[payload.key] != null) return;
      state.logs[payload.key] = { ...ZERO_STATE, key: payload.key };
    },
    setActiveToolbarTab: (
      state,
      { payload }: PayloadAction<SetActiveToolbarTabPayload>,
    ) => {
      const log = state.logs[payload.key];
      if (log != null) log.toolbar.activeTab = payload.tab;
    },
    clearPendingUpload: (
      state,
      { payload }: PayloadAction<ClearPendingUploadPayload>,
    ) => {
      const log = state.logs[payload.key];
      if (log != null) log.pendingUpload = undefined;
    },
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      payload.keys.forEach((key) => delete state.logs[key]);
    },
  },
});

export const {
  create: internalCreate,
  setActiveToolbarTab,
  clearPendingUpload,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export type Payload = Action["payload"];
