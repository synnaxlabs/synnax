// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

import * as latest from "@/cluster/migrations";

export type Cluster = latest.Cluster;
export type SliceState = latest.SliceState;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;
export const LOCAL_CLUSTER_KEY = latest.LOCAL_CLUSTER_KEY;
export const isLocalCluster = latest.isLocalCluster;

export const SLICE_NAME = "cluster";

/**
 * Represents a partial view of a larger store that contains the cluster slice. This is
 * typically used for hooks that accept the entire store state as a parameter but only
 * need access to the cluster slice.
 */
export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const PERSIST_EXCLUDE = `${SLICE_NAME}.localState.status`;

/** Signature for the setCluster action. */
export type SetPayload = Cluster;

/** Signature for the setActiveCluster action. */
export type SetActivePayload = string | null;

export interface RemovePayload {
  keys: string[];
}

export interface RenamePayload {
  key: string;
  name: string;
}

export const {
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
      state.clusters[cluster.key] = cluster;
      state.activeCluster ??= cluster.key;
    },
    remove: ({ clusters }, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      for (const key of keys) delete clusters[key];
    },
    setActive: (state, { payload: key }: PayloadAction<SetActivePayload>) => {
      state.activeCluster = key;
    },
    rename: (state, { payload: { key, name } }: PayloadAction<RenamePayload>) => {
      const cluster = state.clusters[key];
      if (cluster == null) return;
      cluster.name = name;
      if (cluster.props != null) cluster.props.name = name;
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
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export type Payload = Action["payload"];
