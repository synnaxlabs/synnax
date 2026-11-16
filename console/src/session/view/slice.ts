// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { panel, view } from "@synnaxlabs/client";
import z from "zod";

export const SLICE_NAME = "view";

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  /** The saved view each explorer tab shows. A tab with none shows its static view. */
  open: z.record(panel.tabKeyZ, view.keyZ).default({}),
});

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({});

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface SetOpenPayload {
  tab: panel.TabKey;
  /** The saved view to show, or null for the tab's static view. */
  view: view.Key | null;
}

export interface RemoveViewsPayload {
  keys: view.Key[];
}

export interface RemoveTabsPayload {
  keys: panel.TabKey[];
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setOpen: (state, { payload: { tab, view } }: PayloadAction<SetOpenPayload>) => {
      if (view == null) delete state.open[tab];
      else state.open[tab] = view;
    },
    removeViews: (state, { payload: { keys } }: PayloadAction<RemoveViewsPayload>) => {
      const removed = new Set(keys);
      Object.entries(state.open).forEach(([tab, view]) => {
        if (removed.has(view)) delete state.open[tab];
      });
    },
    removeTabs: (state, { payload: { keys } }: PayloadAction<RemoveTabsPayload>) => {
      keys.forEach((tab) => delete state.open[tab]);
    },
  },
});

export const { setOpen, removeViews, removeTabs } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
