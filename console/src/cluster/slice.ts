// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { array } from "@synnaxlabs/x";

import * as latest from "@/cluster/types";

export const clusterZ = latest.clusterZ;
export type Cluster = latest.Cluster;
export type SliceState = latest.SliceState;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;
const getPredefinedClusterKey = latest.getPredefinedClusterKey;

export const SLICE_NAME = "cluster";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type SetPayload = Cluster;

export type SetActivePayload = string | null;

export type RemovePayload = string | string[];

export interface RenamePayload {
  key: string;
  name: string;
}

export interface ChangeKeyPayload {
  oldKey: string;
  newKey: string;
}

const checkName = (state: SliceState, name: string, key?: string) => {
  if (Object.values(state.clusters).some((c) => c.name === name && c.key !== key))
    throw new Error(`A cluster with the name ${name} already exists.`);
};

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    set: (state, { payload: cluster }: PayloadAction<SetPayload>) => {
      const predefinedKey = getPredefinedClusterKey(cluster);
      if (predefinedKey != null) delete state.clusters[predefinedKey];
      state.clusters[cluster.key] = cluster;
    },
    remove: ({ clusters }, { payload: keys }: PayloadAction<RemovePayload>) =>
      array.toArray(keys).forEach((key) => delete clusters[key]),
    setActive: (state, { payload: key }: PayloadAction<SetActivePayload>) => {
      state.activeCluster = key;
    },
    rename: (state, { payload: { key, name } }: PayloadAction<RenamePayload>) => {
      checkName(state, name);
      state.clusters[key].name = name;
    },
    changeKey: (
      state,
      { payload: { oldKey, newKey } }: PayloadAction<ChangeKeyPayload>,
    ) => {
      const cluster = state.clusters[oldKey];
      delete state.clusters[oldKey];
      state.clusters[newKey] = { ...cluster, key: newKey };
      if (state.activeCluster === oldKey) state.activeCluster = newKey;
    },
  },
});

export const { set, setActive, remove, rename, changeKey } = actions;

export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
