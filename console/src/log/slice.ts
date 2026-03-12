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
export const ZERO_CHANNEL_CONFIG = latest.ZERO_CHANNEL_CONFIG;
export const stateZ = latest.stateZ;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const ZERO_STATE = latest.ZERO_STATE;

export const SLICE_NAME = "log";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type CreatePayload = State;

export interface SetChannelsPayload {
  key: string;
  channels: channel.Key[];
}

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
    setChannels: (state, { payload }: PayloadAction<SetChannelsPayload>) => {
      state.logs[payload.key].channels = payload.channels;
    },
    setTimestampPrecision: (
      state,
      { payload }: PayloadAction<SetTimestampPrecisionPayload>,
    ) => {
      state.logs[payload.key].timestampPrecision = payload.timestampPrecision;
    },
    setChannelConfig: (state, { payload }: PayloadAction<SetChannelConfigPayload>) => {
      const logState = state.logs[payload.key];
      const existing =
        logState.channelConfigs[String(payload.channelKey)] ?? ZERO_CHANNEL_CONFIG;
      logState.channelConfigs[String(payload.channelKey)] = {
        ...existing,
        ...payload.config,
      };
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
  setChannels,
  setTimestampPrecision,
  setChannelConfig,
  setRemoteCreated,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export type Payload = Action["payload"];
