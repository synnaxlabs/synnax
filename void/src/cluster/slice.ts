import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { Connectivity, SynnaxProps } from "@synnaxlabs/client";
import { Dispatch } from "react";
import { useSelector } from "react-redux";
import { Optional } from "../util/types";

export type Cluster = {
  key: string;
  name: string;
  props: SynnaxProps;
  active: boolean;
  state: ConnectionState;
};

export type ConnectionState = {
  status: Connectivity;
  message?: string;
};

export type ClusterSliceState = {
  reconnect: boolean;
  clusters: Cluster[];
};

export type ClusterSliceStoreState = {
  cluster: ClusterSliceState;
};

const initialState: ClusterSliceState = {
  reconnect: false,
  clusters: [
    {
      key: "local",
      name: "Synnax",
      props: {
        host: "localhost",
        port: 9090,
      },
      active: true,
      state: { status: Connectivity.DISCNNECTED },
    },
  ],
};

export type SetClusterAction = PayloadAction<Optional<Cluster, "state">>;
export type SetActiveClusterAction = PayloadAction<string>;
export type SetClusterConnectionState = PayloadAction<{
  key: string;
  state: ConnectionState;
}>;
export type ToggleRecconnectAction = PayloadAction<undefined>;

const slice = createSlice({
  name: "cluster",
  initialState,
  reducers: {
    toggleReconnect: (state) => {
      state.reconnect = !state.reconnect;
    },
    setClusterConnectionState: (
      { clusters },
      { payload: { key, state } }: SetClusterConnectionState
    ) => {
      const cluster = clusters.find((c) => c.key === key);
      if (cluster) {
        cluster.state = state;
      }
    },
    setCluster: ({ clusters }, { payload: cluster }: SetClusterAction) => {
      const index = clusters.findIndex(({ key }) => key === cluster.key);
      if (index >= 0) {
        clusters[index] = cluster as Cluster;
      } else {
        clusters.push(cluster as Cluster);
      }
      if (cluster.active) {
        clusters = changeActiveCluster(clusters, cluster.key);
      }
    },
    setActiveCluster: (state, { payload: key }: SetActiveClusterAction) => {
      state.clusters = changeActiveCluster(state.clusters, key);
    },
  },
});

export const useSelectActiveCluster = () =>
  useSelector(({ cluster: { clusters } }: ClusterSliceStoreState) =>
    clusters.find(({ active }) => active)
  );

export const useSelectCluster = (key: string): Cluster | undefined =>
  useSelector(({ cluster: { clusters } }: ClusterSliceStoreState) =>
    clusters.find(({ key: clusterKey }) => clusterKey === key)
  );

const changeActiveCluster = (clusters: Cluster[], key: string) => {
  return clusters.map((cluster) => ({
    ...cluster,
    active: cluster.key === key,
  }));
};

export const {
  toggleReconnect,
  setCluster,
  setActiveCluster,
  setClusterConnectionState,
} = slice.actions;

export default slice;
