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

/** The link between this machine and a Synnax account, held by Synnax Desktop. */
export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  /** The state minted for a sign-in the app started and has not finished. */
  pending: z.string().optional(),
  /** The activation the portal issued for this machine. */
  activation: z.string().optional(),
  /** The secret that renews this machine's license. */
  secret: z.string().optional(),
  /** The address of the account this machine is linked to. */
  email: z.string().optional(),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface LinkPayload {
  activation: string;
  secret: string;
  email: string;
}

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    beginSignIn: (state, { payload }: PayloadAction<string>) => {
      state.pending = payload;
    },
    link: (state, { payload }: PayloadAction<LinkPayload>) => {
      state.pending = undefined;
      state.activation = payload.activation;
      state.secret = payload.secret;
      state.email = payload.email;
    },
    clear: () => ZERO_SLICE_STATE,
  },
});

export const { beginSignIn, link, clear } = actions;
export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
