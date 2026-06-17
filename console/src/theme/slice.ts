// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { Theming } from "@synnaxlabs/pluto";
import { z } from "zod";

export const SLICE_NAME = "theme";

export const sliceStateZ = z.object({
  active: z.string(),
  themes: z.record(z.string(), Theming.themeZ),
});

export interface SliceState extends z.output<typeof sliceStateZ> {}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({
  active: Theming.SYNNAX_DARK.key,
  themes: {
    [Theming.SYNNAX_DARK.key]: Theming.SYNNAX_THEMES.synnaxDark,
    [Theming.SYNNAX_LIGHT.key]: Theming.SYNNAX_THEMES.synnaxLight,
  },
});

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setActive: (state, { payload: key }: PayloadAction<string | undefined>) => {
      if (key != null) state.active = key;
      else {
        const keys = Object.keys(state.themes).sort();
        const index = keys.indexOf(state.active);
        state.active = keys[(index + 1) % keys.length];
      }
    },
    toggle: (state) => {
      const keys = Object.keys(state.themes);
      const index = keys.indexOf(state.active);
      state.active = keys[(index + 1) % keys.length];
    },
  },
});

export const { setActive, toggle } = actions;

export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];
