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

/**
 * Represents a partial view of a larger store that contains the cluster slice. This is
 * typically used for hooks that accept the entire store state as a parameter but only
 * need access to the cluster slice.
 */
export interface StoreState {
  [SLICE_NAME]: SliceState;
}

/** Signature for the setCluster action. */
export type SetPayload = Cluster;

/** Signature for the setActiveCluster action. */
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

const checkName = (state: SliceState, name: string) => {
  // if (Object.values(state.clusters).some((c) => c.name === name))
  //   throw new Error(`A cluster with the name ${name} already exists.`);
};

const {
  actions,
  /**
   * The reducer for the cluster slice.
   */
  reducer,
} = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    set: (state, { payload: cluster }: PayloadAction<SetPayload>) => {
      checkName(state, cluster.name);
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

export const {
  /**
   * Sets the cluster with the given key in state.
   * @params payload.cluster - The cluster to set.
   */
  set,
  /**
   * Sets the active cluster key in state.
   * @params payload - The key of the cluster to set as active.
   */
  setActive,
  remove,
  rename,
  changeKey,
} = actions;

export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
