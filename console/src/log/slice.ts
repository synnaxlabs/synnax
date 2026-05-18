// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type channel } from "@synnaxlabs/client";

import * as latest from "@/log/types";

export type State = latest.State;
export type SliceState = latest.SliceState;
export type ChannelConfig = latest.ChannelConfig;
export type ChannelEntry = latest.ChannelEntry;
export type ToolbarTab = latest.ToolbarTab;
export type ToolbarState = latest.ToolbarState;
export const ZERO_CHANNEL_CONFIG = latest.ZERO_CHANNEL_CONFIG;
export const ZERO_CHANNEL_ENTRY = latest.ZERO_CHANNEL_ENTRY;
export const ZERO_TOOLBAR_STATE = latest.ZERO_TOOLBAR_STATE;
export const stateZ = latest.stateZ;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const ZERO_STATE = latest.ZERO_STATE;
export const migrateSlice = latest.migrateSlice;

export const SLICE_NAME = "log";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type CreatePayload = State;

export interface SetTimestampPrecisionPayload {
  key: string;
  timestampPrecision: number;
}

export interface SetChannelConfigPayload {
  key: string;
  channelKey: channel.Key;
  config: Partial<ChannelConfig>;
}

export interface SetRemoteCreatedPayload {
  key: string;
}

export interface SetShowChannelNamesPayload {
  key: string;
  showChannelNames: boolean;
}

export interface SetShowReceiptTimestampPayload {
  key: string;
  showReceiptTimestamp: boolean;
}

export interface SetActiveToolbarTabPayload {
  key: string;
  tab: ToolbarTab;
}

export interface AddChannelPayload {
  key: string;
  channelKey: channel.Key;
}

export interface RemoveChannelByIndexPayload {
  key: string;
  index: number;
}

export interface SetChannelAtIndexPayload {
  key: string;
  index: number;
  channelKey: channel.Key;
}

export interface RemovePayload {
  keys: string[];
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: latest.ZERO_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      const { key } = payload;
      state.logs[key] = payload;
    },
    setTimestampPrecision: (
      state,
      { payload }: PayloadAction<SetTimestampPrecisionPayload>,
    ) => {
      state.logs[payload.key].timestampPrecision = payload.timestampPrecision;
    },
    setChannelConfig: (state, { payload }: PayloadAction<SetChannelConfigPayload>) => {
      const logState = state.logs[payload.key];
      const entry = logState.channels.find((e) => e.channel === payload.channelKey);
      if (entry != null) Object.assign(entry, payload.config);
    },
    setShowChannelNames: (
      state,
      { payload }: PayloadAction<SetShowChannelNamesPayload>,
    ) => {
      state.logs[payload.key].showChannelNames = payload.showChannelNames;
    },
    setShowReceiptTimestamp: (
      state,
      { payload }: PayloadAction<SetShowReceiptTimestampPayload>,
    ) => {
      state.logs[payload.key].showReceiptTimestamp = payload.showReceiptTimestamp;
    },
    setActiveToolbarTab: (
      state,
      { payload }: PayloadAction<SetActiveToolbarTabPayload>,
    ) => {
      state.logs[payload.key].toolbar.activeTab = payload.tab;
    },
    addChannel: (state, { payload }: PayloadAction<AddChannelPayload>) => {
      state.logs[payload.key].channels.push({
        ...ZERO_CHANNEL_CONFIG,
        channel: payload.channelKey,
      });
    },
    removeChannelByIndex: (
      state,
      { payload }: PayloadAction<RemoveChannelByIndexPayload>,
    ) => {
      state.logs[payload.key].channels.splice(payload.index, 1);
    },
    setChannelAtIndex: (
      state,
      { payload }: PayloadAction<SetChannelAtIndexPayload>,
    ) => {
      const entry = state.logs[payload.key].channels[payload.index];
      if (entry != null) entry.channel = payload.channelKey;
    },
    setRemoteCreated: (state, { payload }: PayloadAction<SetRemoteCreatedPayload>) => {
      state.logs[payload.key].remoteCreated = true;
    },
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      payload.keys.forEach((key) => delete state.logs[key]);
    },
  },
});

export const {
  create: internalCreate,
  setTimestampPrecision,
  setChannelConfig,
  setShowChannelNames,
  setShowReceiptTimestamp,
  setActiveToolbarTab,
  addChannel,
  removeChannelByIndex,
  setChannelAtIndex,
  setRemoteCreated,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export type Payload = Action["payload"];
