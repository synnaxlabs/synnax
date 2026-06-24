// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type log } from "@synnaxlabs/client";
import z from "zod";

export const toolbarTabZ = z.enum(["channels", "properties"]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;

export const toolbarStateZ = z.object({
  selectedTab: toolbarTabZ.default("channels"),
});
export interface ToolbarState extends z.infer<typeof toolbarStateZ> {}

export const stateZ = z.object({
  toolbar: toolbarStateZ.prefault({}),
});
export interface State extends z.infer<typeof stateZ> {}
export interface NewState extends z.input<typeof stateZ> {}

export const ZERO_STATE = stateZ.parse({});

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  logs: z.record(z.string(), stateZ).default({}),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "log";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface KeyedPayload {
  key: log.Key;
}

export interface CreatePayload extends KeyedPayload, NewState {}

export interface SetActiveToolbarTabPayload extends KeyedPayload {
  tab: ToolbarTab;
}

export interface RemovePayload {
  keys: string[];
}

const withSelectedState =
  <Payload extends KeyedPayload, Type extends string = string>(
    handler?: (state: State, action: PayloadAction<Payload, Type>) => void,
  ) =>
  (state: SliceState, action: PayloadAction<Payload, Type>) => {
    const {
      payload: { key },
    } = action;
    let s = state.logs[key];
    if (s == null) {
      s = stateZ.parse({});
      state.logs[key] = s;
    }
    handler?.(s, action);
  };

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      if (payload.key in state.logs) return;
      state.logs[payload.key] = stateZ.parse(payload);
    },
    setActiveToolbarTab: withSelectedState(
      (state, { payload: { tab } }: PayloadAction<SetActiveToolbarTabPayload>) => {
        state.toolbar.selectedTab = tab;
      },
    ),
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      payload.keys.forEach((key) => delete state.logs[key]);
    },
  },
});

export const { create: internalCreate, setActiveToolbarTab, remove } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const PERSIST_EXCLUDE = [];
