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

import * as latest from "@/spectrogram/types";

export const SLICE_NAME = "spectrogram";

export type State = latest.State;
export type ToolbarTab = latest.ToolbarTab;
export type ToolbarState = latest.ToolbarState;
export type SliceState = latest.SliceState;
export const ZERO_STATE = latest.ZERO_STATE;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;

export const migrateSlice = (v: unknown): SliceState => v as SliceState;

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface CreatePayload {
  key: string;
}

export interface RemovePayload {
  keys: string[];
}

export interface SetChannelPayload {
  key: string;
  channel: channel.Key;
}

export interface SetSampleRatePayload {
  key: string;
  sampleRate: number;
}

export interface SetFFTParamsPayload {
  key: string;
  fftSize?: number;
  windowFunction?: "hann" | "blackmanHarris";
  overlap?: number;
}

export interface SetDisplayPayload {
  key: string;
  colorMap?: "viridis" | "inferno" | "magma" | "plasma" | "jet" | "grayscale";
  dbMin?: number;
  dbMax?: number;
  freqMin?: number;
  freqMax?: number;
}

export interface SetActiveToolbarTabPayload {
  key: string;
  tab: ToolbarTab;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: latest.ZERO_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      if (state.spectrograms[payload.key] != null) return;
      state.spectrograms[payload.key] = {
        ...latest.ZERO_STATE,
        key: payload.key,
      };
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      keys.forEach((k) => {
        delete state.spectrograms[k];
      });
    },
    setChannel: (state, { payload }: PayloadAction<SetChannelPayload>) => {
      const s = state.spectrograms[payload.key];
      if (s != null) s.channel = payload.channel;
    },
    setSampleRate: (state, { payload }: PayloadAction<SetSampleRatePayload>) => {
      const s = state.spectrograms[payload.key];
      if (s != null) s.sampleRate = payload.sampleRate;
    },
    setFFTParams: (state, { payload }: PayloadAction<SetFFTParamsPayload>) => {
      const s = state.spectrograms[payload.key];
      if (s == null) return;
      if (payload.fftSize != null) s.fftSize = payload.fftSize;
      if (payload.windowFunction != null) s.windowFunction = payload.windowFunction;
      if (payload.overlap != null) s.overlap = payload.overlap;
    },
    setDisplay: (state, { payload }: PayloadAction<SetDisplayPayload>) => {
      const s = state.spectrograms[payload.key];
      if (s == null) return;
      if (payload.colorMap != null) s.colorMap = payload.colorMap;
      if (payload.dbMin != null) s.dbMin = payload.dbMin;
      if (payload.dbMax != null) s.dbMax = payload.dbMax;
      if (payload.freqMin != null) s.freqMin = payload.freqMin;
      if (payload.freqMax != null) s.freqMax = payload.freqMax;
    },
    setActiveToolbarTab: (
      state,
      { payload }: PayloadAction<SetActiveToolbarTabPayload>,
    ) => {
      const s = state.spectrograms[payload.key];
      if (s != null) s.toolbar.activeTab = payload.tab;
    },
  },
});

export const {
  create: internalCreate,
  remove,
  setChannel,
  setSampleRate,
  setFFTParams,
  setDisplay,
  setActiveToolbarTab,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
