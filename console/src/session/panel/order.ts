// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { panel } from "@synnaxlabs/client";
import { array, compare } from "@synnaxlabs/x";
import z from "zod";

import { remove, reset } from "@/session/panel/slice";

export const ORDER_SLICE_NAME = "panelOrder";

/**
 * The strip's panel order, shared by every window and owned by the project rather than
 * by any one view of it.
 */
export const orderSliceStateZ = z.object({
  version: z.literal(0).default(0),
  order: panel.keyZ.array().default([]),
});

export interface OrderSliceState extends z.output<typeof orderSliceStateZ> {}

export const ZERO_ORDER_SLICE_STATE: OrderSliceState = orderSliceStateZ.parse({});

export interface OrderStoreState {
  [ORDER_SLICE_NAME]: OrderSliceState;
}

export interface OrderEntry {
  key: panel.Key;
  name: string;
}

export interface ReconcileOrderPayload {
  panels: OrderEntry[];
}

export interface ReorderPayload {
  key: panel.Key;
  index: number;
}

const { actions, reducer: orderReducer } = createSlice({
  name: ORDER_SLICE_NAME,
  initialState: ZERO_ORDER_SLICE_STATE,
  reducers: {
    // reconcileOrder converges the order to the project's live membership: deleted
    // panels prune, unknown panels append name-sorted. On first sight of a project the
    // whole order materializes name-sorted through the same append path.
    reconcileOrder: (
      state,
      { payload: { panels } }: PayloadAction<ReconcileOrderPayload>,
    ) => {
      const live = new Set(panels.map(({ key }) => key));
      const kept = state.order.filter((key) => live.has(key));
      const fresh = panels
        .filter(({ key }) => !kept.includes(key))
        .sort((a, b) => compare.stringsWithNumbers(a.name, b.name))
        .map(({ key }) => key);
      const next = [...kept, ...fresh];
      if (!compare.arraysEqual(state.order, next)) state.order = next;
    },
    // The index is a strip insertion slot resolved with the panel still in place, so a
    // move toward the end lands one slot short of the raw index.
    reorder: (state, { payload: { key, index } }: PayloadAction<ReorderPayload>) => {
      const from = state.order.indexOf(key);
      const next = [...state.order];
      if (from !== -1) next.splice(from, 1);
      const to = Math.min(from !== -1 && from < index ? index - 1 : index, next.length);
      next.splice(Math.max(to, 0), 0, key);
      if (!compare.arraysEqual(state.order, next)) state.order = next;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(remove, (state, { payload: { keys } }) => {
        const removed = array.toArray(keys);
        state.order = state.order.filter((key) => !removed.includes(key));
      })
      .addCase(reset, () => ZERO_ORDER_SLICE_STATE);
  },
});

export const { reconcileOrder, reorder } = actions;

export { orderReducer };

export type OrderAction =
  | ReturnType<(typeof actions)[keyof typeof actions]>
  | ReturnType<typeof remove>
  | ReturnType<typeof reset>;
