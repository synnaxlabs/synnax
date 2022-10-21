import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { Connectivity, SynnaxProps } from "@synnaxlabs/client";
import { useSelector } from "react-redux";

export type Cluster = {
  key: string;
  name: string;
  props: SynnaxProps;
  active: boolean;
  state: ConnectionState;
};

export type ConnectionState = {
  status: Connectivity;
  message: string;
};

type ClusterSliceState = {
  clusters: Cluster[];
};

type StoreState = {
  cluster: ClusterSliceState;
};

const initialState: ClusterSliceState = {
  clusters: [],
};

export type SetClusterAction = PayloadAction<Cluster>;
export type SetActiveClusterAction = PayloadAction<string | Cluster>;

const slice = createSlice({
  name: "cluster",
  initialState,
  reducers: {
    setCluster: ({ clusters }, { payload: cluster }) => {
      cluster.key = clusterKey(cluster);
      const index = clusters.findIndex(({ key }) => key === cluster);
      if (index >= 0) {
        clusters[index] = cluster;
      } else {
        clusters.push(cluster);
      }
      if (cluster.active) {
        clusters = changeActiveCluster(clusters, cluster);
      }
    },
    setActiveCluster: (state, { payload: keyOrCluster }) => {
      state.clusters = changeActiveCluster(state.clusters, keyOrCluster);
    },
  },
});

export const useSelectActiveCluster = () =>
  useSelector(({ cluster: { clusters } }: StoreState) =>
    clusters.find(({ active }) => active)
  );

const clusterKey = ({ props: { host, port } }: Cluster) => `${host}:${port}`;

const changeActiveCluster = (clusters: Cluster[], key: Cluster | string) => {
  return clusters.map((cluster) => ({
    ...cluster,
    active:
      cluster.key === (typeof key === "string" ? key : clusterKey(cluster)),
  }));
};

export const { setCluster, setActiveCluster } = slice.actions;
export default slice;
