import memoize from "proxy-memoize";
import { useCallback } from "react";
import { useSelector } from "react-redux";
import { Cluster } from "../types";
import { ClusterStoreState } from "./slice";

export const useSelectActiveCluster = (): Cluster | undefined =>
  useSelector(
    useCallback(
      memoize(
        (state: ClusterStoreState) =>
          state.cluster.clusters[state.cluster.activeClusterKey || ""]
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

export const useSelectCluster = (key: string | undefined) =>
  useSelector(
    useCallback(
      memoize((state: ClusterStoreState) => state.cluster.clusters[key || ""]),
      [key]
    )
  );

export const useSelectClusters = () => {
  return useSelector(
    useCallback(
      memoize((state: ClusterStoreState) => state.cluster.clusters),
      []
    )
  );
};
