// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice } from "@reduxjs/toolkit";
import type { PayloadAction } from "@reduxjs/toolkit";
import { Optional } from "@synnaxlabs/x";

import { Cluster, ConnectionState, DEFAULT_CONNECTION_STATE } from "@/cluster/core";

/** The state of the cluster slice. */
export interface ClusterState {
  /** The current, active cluster. */
  activeCluster: string | null;
  /**
   * A record of cluster keys to clusters. The active cluster is guaranteed
   * to be present in this record.
   */
  clusters: Record<string, Cluster>;
}

/**
 * The name of the cluster slice in a larger store.
 * NOTE: This must be the name of the slice in the store, or else all selectors will fail.
 */
export const CLUSTER_SLICE_NAME = "cluster";

/**
 * Represents a partial view of a larger store that contains the cluster slice. This is
 * typically used for hooks that accept the entire store state as a parameter but only
 * need access to the cluster slice.
 */
export interface ClusterStoreState {
  [CLUSTER_SLICE_NAME]: ClusterState;
}

const initialState: ClusterState = {
  activeCluster: null,
  clusters: {},
};

/** Signature for the setCluster action. */
export type SetClusterAction = PayloadAction<Optional<Cluster, "state">>;
/** Signature for the setActiveCluster action. */
export type SetActiveClusterAction = PayloadAction<string | null>;
/** Signature for the setClusterConnectionState action. */
export type SetClusterConnectionState = PayloadAction<{
  key: string;
  state: ConnectionState;
}>;

export const {
  actions: {
    /**
     * Sets the cluster with the given key in state.
     * @params payload.cluster - The cluster to set.
     */
    setCluster,
    /**
     * Sets the active cluster key in state.
     * @params payload - The key of the cluster to set as active.
     */
    setActiveCluster,
    /**
     * Sets the connection state of the cluster with the given key in state.
     * @params payload.key - The key of the cluster to set the connection state of.
     * @params payload.state - The connection state to set.
     */
    setClusterConnectionState,
  },
  /**
   * The reducer for the cluster slice.
   */
  reducer: clusterReducer,
} = createSlice({
  name: CLUSTER_SLICE_NAME,
  initialState,
  reducers: {
    setClusterConnectionState: (
      { clusters },
      { payload: { key, state } }: SetClusterConnectionState
    ) => {
      clusters[key].state = state;
    },
    setCluster: ({ clusters }, { payload: cluster }: SetClusterAction) => {
      clusters[cluster.key] = { state: DEFAULT_CONNECTION_STATE, ...cluster };
    },
    setActiveCluster: (state, { payload: key }: SetActiveClusterAction) => {
      state.activeCluster = key;
    },
  },
});
