// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createSlice,
  type Dispatch,
  type PayloadAction,
  type UnknownAction,
} from "@reduxjs/toolkit";
import { type policy, Synnax, user } from "@synnaxlabs/client";

import * as latest from "@/permissions/migrations";

export type SliceState = latest.SliceState;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;

export const SLICE_NAME = "permissions";

export type StoreState = {
  [SLICE_NAME]: SliceState;
};

export interface ClearPayload {}

export interface SetPayload {
  policies: policy.Policy[];
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    clear: (state) => void (state.policies = []),
    set: (state, { payload: { policies } }: PayloadAction<SetPayload>) =>
      void (state.policies = policies),
  },
});

export const { clear, set } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];

export const setCurrentUserPermissions = async (
  client: Synnax | null,
  dispatch: Dispatch<UnknownAction>,
): Promise<void> => {
  console.log("setCurrentUserPermissions");
  dispatch(clear());
  if (client == null) {
    // TODO: give current user all permissions?
    return;
  }
  const username = client.props.username;
  const user_ = await client.user.retrieveByName(username);
  console.log("user_", user_);
  const policies = await client.access.policy.retrieveFor(user.ontologyID(user_.key));
  console.log("policies", policies);
  dispatch(set({ policies }));
};
