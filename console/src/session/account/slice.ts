// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { z } from "zod";

export const SLICE_NAME = "account";

/** The account this machine is linked to. */
const linkZ = z.object({
  /** The secret that renews this machine's license. */
  secret: z.string(),
  /** The address of the account. */
  email: z.string(),
  /** The Clerk ID of the account. */
  user: z.string(),
});
export interface Link extends z.infer<typeof linkZ> {}

/** A login link the hub sent, held until the Core applies its license key. */
const receivedZ = linkZ.extend({
  key: z.string(),
  /** True when the person allowed usage data on the hub. */
  analyticsAllowed: z.boolean(),
});
export interface Received extends z.infer<typeof receivedZ> {}

/** The link between this machine and a Synnax account, held by Synnax Desktop. */
export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  /** The state minted for a login the app started and has not finished. */
  pending: z.string().optional(),
  received: receivedZ.optional(),
  link: linkZ.optional(),
  /** True when the person allows Synnax Desktop to send usage data. */
  analyticsAllowed: z.boolean().default(false),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

/** A new link, with the usage data choice the person made on the hub. */
export interface LinkPayload extends Link {
  analyticsAllowed: boolean;
}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    beginLogin: (state, { payload }: PayloadAction<string>) => {
      state.pending = payload;
    },
    receive: (state, { payload }: PayloadAction<Received>) => {
      state.received = payload;
    },
    discardReceived: (state) => {
      state.received = undefined;
    },
    link: (
      state,
      { payload: { analyticsAllowed, ...next } }: PayloadAction<LinkPayload>,
    ) => {
      state.pending = undefined;
      state.received = undefined;
      state.link = next;
      state.analyticsAllowed = analyticsAllowed;
    },
    setAnalyticsAllowed: (state, { payload }: PayloadAction<boolean>) => {
      state.analyticsAllowed = payload;
    },
    clear: () => ZERO_SLICE_STATE,
  },
});

export const {
  beginLogin,
  receive,
  discardReceived,
  link,
  setAnalyticsAllowed,
  clear,
} = actions;
export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
