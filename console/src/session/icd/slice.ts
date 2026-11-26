// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import type { icd } from "@synnaxlabs/client";
import type { Drift } from "@synnaxlabs/drift";
import z from "zod";

import { Window } from "@/session/window";

const stateZ = z.object({
  /** The keys of the selected messages. The editor shows the first. */
  selected: z.array(z.string()).default([]),
  /** The field that the editor focuses once it shows it. The editor then clears it. */
  focused: z.string().optional(),
});
export interface State extends z.infer<typeof stateZ> {}

export const ZERO_STATE = stateZ.parse({});

export interface SliceState {
  /** A window is a viewport, so each holds its own view of every ICD. */
  windows: Record<string, Window.Documents<State>>;
}

export const ZERO_SLICE_STATE: SliceState = { windows: {} };

export const SLICE_NAME = "icd";

export interface StoreState extends Drift.StoreState {
  [SLICE_NAME]: SliceState;
}

export interface SetSelectedPayload extends Window.OptionalKeyParams {
  key: icd.Key;
  selected: icd.MessageKey[];
  /** A field of the first selected message for the editor to focus. */
  focused?: icd.FieldKey;
}

export interface ClearFocusedPayload extends Window.OptionalKeyParams {
  key: icd.Key;
}

export interface RemovePayload {
  keys: icd.Key[];
}

const withSelectedState = Window.createWithDocumentHandler(stateZ);

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setSelected: withSelectedState<SetSelectedPayload, SliceState>(
      (state, { payload: { selected, focused } }) => {
        state.selected = selected;
        state.focused = focused;
      },
    ),
    clearFocused: withSelectedState<ClearFocusedPayload, SliceState>((state) => {
      delete state.focused;
    }),
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      Window.removeDocuments(state, payload.keys);
    },
  },
  extraReducers: Window.handleRemoved,
});

export const { setSelected, clearFocused, remove } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const MIDDLEWARE = [
  Window.createInjectKeyMiddleware([setSelected, clearFocused]),
];
