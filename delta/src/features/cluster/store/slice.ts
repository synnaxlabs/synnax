import { createSlice } from "@reduxjs/toolkit";
import type { PayloadAction } from "@reduxjs/toolkit";

import { Optional } from "../../../util/types";
import { Cluster, ConnectionState, DEFAULT_CONNECTION_STATE } from "../types";

/** Represents the state of the cluster slice. */
export interface ClusterSliceState {
  /** The current, active cluster. */
  activeClusterKey: string | null;
  /** A record of cluster keys to clusters. The key activeClusterKey is guaranteed
   * to be present in this record.
   */
  clusters: Record<string, Cluster>;
}

/**
 * The name of the cluster slice in a larger store.
 * NOTE: This must be the name of the slice in the store, or else all selectors will fail.
 */
export const CLUSTER_SLICE_NAME = "cluster";

/** Represents a partial view of a large store that contains the cluster slice. */
export interface ClusterStoreState {
  [CLUSTER_SLICE_NAME]: ClusterSliceState;
}

const devClusterProps = {
  key: "dev",
  name: "Development",
  props: {
    host: "localhost",
    port: 9090,
    username: "synnax",
    password: "seldon",
  },
  state: DEFAULT_CONNECTION_STATE,
};

const initialState: ClusterSliceState = {
  activeClusterKey: "dev",
  clusters: {
    dev: devClusterProps,
  },
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
  name: "cluster",
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
      state.activeClusterKey = key;
    },
  },
});
