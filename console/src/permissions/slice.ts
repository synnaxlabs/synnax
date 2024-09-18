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
import { Synnax } from "@synnaxlabs/client";
import { migrate } from "@synnaxlabs/x";

import {
  allowAllPolicy,
  consolePolicyMap,
  consolePolicySet,
  initialPermissions,
  type Permissions,
  policiesAreEqual,
} from "@/permissions/permissions";

export const SLICE_NAME = "permissions";

export interface SliceState extends migrate.Migratable {
  permissions: Permissions;
}

export type StoreState = {
  [SLICE_NAME]: SliceState;
};

const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.0",
  permissions: { ...initialPermissions },
};

const MIGRATIONS = {};

export const migrateSlice = migrate.migrator<SliceState, SliceState>({
  name: "user.slice",
  migrations: MIGRATIONS,
  def: ZERO_SLICE_STATE,
});

interface SetPayload {
  permissions: Permissions;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    giveAll: (state) =>
      consolePolicySet.forEach((policy) => (state.permissions[policy] = true)),
    removeAll: (state) =>
      consolePolicySet.forEach((policy) => (state.permissions[policy] = false)),
    set: (state, { payload }: PayloadAction<SetPayload>) => {
      const { permissions } = payload;
      state.permissions = { ...permissions };
    },
  },
});

export const { giveAll, removeAll, set } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const setCurrentUserPermissions = async (
  client: Synnax | null,
  dispatch: Dispatch<UnknownAction>,
): Promise<void> => {
  //This is super jank - we need to wait for when we expect the server to be
  //authenticated before we can get the key of the current user using the client
  await client?.auth?.authenticating;
  await new Promise((resolve) => setTimeout(resolve, 1000));
  const key = client?.auth?.user?.key;
  dispatch(removeAll());
  if (key == null || client == null) return;

  const policies = await client.access.retrieveFor({ type: "user", key });
  const permissions = { ...initialPermissions };

  consolePolicySet.forEach((consolePolicy) => {
    permissions[consolePolicy] = policies.some((policy) =>
      policiesAreEqual(policy, consolePolicyMap[consolePolicy]),
    );
  });

  dispatch(set({ permissions }));

  if (policies.some((policy) => policiesAreEqual(policy, allowAllPolicy)))
    dispatch(giveAll());
};
