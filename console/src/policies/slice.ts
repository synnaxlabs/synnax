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
import { type access, Synnax, user } from "@synnaxlabs/client";

import * as latest from "@/policies/migrations";

export type SliceState = latest.SliceState;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;

export const SLICE_NAME = "policies";

export type StoreState = {
  [SLICE_NAME]: SliceState;
};

export interface ClearPayload {}

export interface SetPayload {
  policies: access.Policy[];
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

export const setCurrentUserPermissions = async (
  client: Synnax | null,
  dispatch: Dispatch<UnknownAction>,
): Promise<void> => {
  //This is super jank - we need to wait for when we expect the server to be
  //authenticated before we can get the key of the current user using the client
  await new Promise((resolve) => setTimeout(resolve, 1000));
  const key = client?.auth?.user?.key;
  dispatch(clear());
  if (key == null || client == null) return;

  const policies = await client.access.retrieveFor({ type: user.ONTOLOGY_TYPE, key });

  dispatch(set({ policies }));
};
