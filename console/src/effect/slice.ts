// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

import * as latest from "@/effect/types";

export type SliceState = latest.SliceState;
export type Effect = latest.Effect;
export const migrateSlice = latest.migrateSlice;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;

export const SLICE_NAME = "effect";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

interface AddPayload {
  effects: Effect[];
}

interface RemovePayload {
  keys: string[];
}

interface RenamePayload {
  key: string;
  name: string;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    add: (state, { payload: { effects } }: PayloadAction<AddPayload>) => {
      effects.forEach((effect) => {
        state.effects[effect.key] = effect;
      });
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      keys.forEach((key) => delete state.effects[key]);
    },
    rename: (state, { payload: { key, name } }: PayloadAction<RenamePayload>) => {
      const effect = state.effects[key];
      if (effect == null) return;
      effect.name = name;
    },
  },
});

export const { add, remove, rename } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
