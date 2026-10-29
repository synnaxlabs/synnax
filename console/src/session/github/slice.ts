// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import z from "zod";

export const SLICE_NAME = "github";

/** Whether the GitHub CLI can star Synnax, unchecked until the CLI answers. */
export const cliZ = z.enum(["unchecked", "ready", "unavailable"]);
export type CLI = z.infer<typeof cliZ>;

export const sliceStateZ = z
  .object({
    version: z.literal(0).default(0),
    /** True once the user starred Synnax on GitHub, or opened its page to star it. */
    starred: z.boolean().default(false),
    /** Checked again on each launch, so never persisted. */
    cli: cliZ.default("unchecked"),
  })
  .prefault({});

export interface SliceState extends z.output<typeof sliceStateZ> {}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({});

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    markStarred: (state) => {
      state.starred = true;
    },
    setCLI: (state, { payload }: PayloadAction<CLI>) => {
      state.cli = payload;
    },
  },
});

export const { markStarred, setCLI } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

const purgeSliceState = <S extends StoreState>(state: S): S => {
  state[SLICE_NAME].cli = "unchecked";
  return state;
};

export const PERSIST_EXCLUDE = [purgeSliceState];
