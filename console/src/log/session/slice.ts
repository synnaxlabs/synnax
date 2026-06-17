// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { Log } from "@synnaxlabs/pluto";
import z from "zod";

import { Keyed } from "@/keyed";

export const toolbarTabZ = z.enum(["channels", "properties"]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;

export const toolbarStateZ = z.object({
  activeTab: toolbarTabZ.default("channels"),
});
export interface ToolbarState extends z.infer<typeof toolbarStateZ> {}

export const stateZ = z.object({
  toolbar: toolbarStateZ.prefault({}),
});
export interface State extends z.infer<typeof stateZ> {}
export interface NewState extends z.input<typeof stateZ> {}

export const sliceStateZ = z.object({
  logs: z.record(z.string(), stateZ).prefault({}),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const SLICE_NAME = "log";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface CreatePayload extends Keyed.Payload, NewState {}

export interface SetActiveToolbarTabPayload extends Keyed.Payload {
  tab: ToolbarTab;
}

export interface RemovePayload {
  keys: string[];
}

const withSelectedState =
  <Payload extends Keyed.Payload, Type extends string = string>(
    handler: (state: State, action: PayloadAction<Payload, Type>) => void,
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
    handler(s, action);
  };

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: sliceStateZ.parse({}),
  reducers: {
    create: (state, { payload: { key, ...rest } }: PayloadAction<CreatePayload>) => {
      if (!(key in state.logs)) state.logs[key] = stateZ.parse(rest);
    },
    setActiveToolbarTab: withSelectedState(
      (state, { payload: { tab } }: PayloadAction<SetActiveToolbarTabPayload>) => {
        state.toolbar.activeTab = tab;
      },
    ),
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      keys.forEach((key) => delete state.logs[key]);
    },
  },
});

export const { create: internalCreate, setActiveToolbarTab, remove } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).logs[key] ?? stateZ.parse({});

export const selectToolbar = (state: StoreState, key: string): ToolbarState =>
  select(state, key).toolbar;

export const selectActiveToolbarTab = (state: StoreState, key: string): ToolbarTab =>
  selectToolbar(state, key).activeTab;

export const { useSelectState, useSelectToolbar, useSelectActiveToolbarTab } =
  Keyed.createSelectors(Log.useKey, {
    useSelectState: select,
    useSelectToolbar: selectToolbar,
    useSelectActiveToolbarTab: selectActiveToolbarTab,
  });

export const { useSetActiveToolbarTab } = Keyed.createDispatchHandlers(Log.useKey, {
  useSetActiveToolbarTab: ["tab", setActiveToolbarTab],
});
