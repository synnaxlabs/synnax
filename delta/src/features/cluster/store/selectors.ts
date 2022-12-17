import { useCallback } from "react";

import memoize from "proxy-memoize";
import { useSelector } from "react-redux";

import { Cluster } from "../types";

import { ClusterStoreState } from "./slice";

export const useSelectActiveCluster = (): Cluster | undefined =>
  useSelector(
    useCallback(
      memoize(
        (state: ClusterStoreState) =>
          state.cluster.clusters[state.cluster.activeClusterKey ?? ""]
      ),
      []
    )
  );

export const useSelectActiveClusterKey = (): string | null =>
  useSelector(
    useCallback(
      memoize((state: ClusterStoreState) => state.cluster.activeClusterKey),
      []
    )
  );

export const useSelectCluster = (key: string | undefined): Cluster =>
  useSelector(
    useCallback(
      memoize((state: ClusterStoreState) => state.cluster.clusters[key ?? ""]),
      [key]
    )
  );

export const useSelectClusters = (): Record<string, Cluster> => {
  return useSelector(
    useCallback(
      memoize((state: ClusterStoreState) => state.cluster.clusters),
      []
    )
  );
};
