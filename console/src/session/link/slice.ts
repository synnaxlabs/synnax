// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

export const SLICE_NAME = "link";

/**
 * Transient deep-link progress. A pending link belongs to the launch that received it.
 */
export interface SliceState {
  /** Whether a received link waits on a project selection before it can open. */
  awaitingProject: boolean;
  /** An invite code a link brought, which the login surface fills in. */
  invite: Invite | null;
  /** How many invites links have brought, so each one starts a fresh login surface. */
  invitesReceived: number;
}

export interface Invite {
  /** The key of the Core the invite is for. */
  core: string;
  code: string;
}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = {
  awaitingProject: false,
  invite: null,
  invitesReceived: 0,
};

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    beginProjectWait: (state) => {
      state.awaitingProject = true;
    },
    endProjectWait: (state) => {
      state.awaitingProject = false;
    },
    receiveInvite: (state, { payload }: PayloadAction<Invite>) => {
      state.invite = payload;
      state.invitesReceived += 1;
    },
    clearInvite: (state) => {
      state.invite = null;
    },
  },
});

export const { beginProjectWait, endProjectWait, receiveInvite, clearInvite } = actions;
export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
