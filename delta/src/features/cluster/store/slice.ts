import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { Optional } from "../../../util/types";
import { Cluster, ConnectionState, DEFAULT_CONNECTION_STATE } from "../types";

export type ClusterSliceState = {
  activeClusterKey: string | null;
  clusters: Record<string, Cluster>;
};

export type ClusterStoreState = {
  cluster: ClusterSliceState;
};

const initialState: ClusterSliceState = {
  activeClusterKey: "dev",
  clusters: {
    dev: {
      key: "dev",
      name: "Development",
      props: {
        host: "localhost",
        port: 9090,
        username: "synnax",
        password: "seldon",
      },
      state: DEFAULT_CONNECTION_STATE,
    },
  },
};

export type SetClusterAction = PayloadAction<Optional<Cluster, "state">>;
export type SetActiveClusterAction = PayloadAction<string | null>;
export type SetClusterConnectionState = PayloadAction<{
  key: string;
  state: ConnectionState;
}>;

export const {
  actions: { setCluster, setActiveCluster, setClusterConnectionState },
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
