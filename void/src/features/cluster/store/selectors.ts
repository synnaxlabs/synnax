import memoize from "proxy-memoize";
import { useCallback } from "react";
import { useSelector } from "react-redux";
import { Cluster } from "@/features/cluster/types";
import { ClusterStoreState } from "./slice";

export const useSelectActiveCluster = () =>
  useSelector(
    useCallback(
      memoize((state: ClusterStoreState) =>
        state.cluster.clusters.find((c) => c.active)
      ),
      []
    )
  );

export const useSelectCluster = (key: string) =>
  useSelector(
    useCallback(
      memoize((state: ClusterStoreState) =>
        state.cluster.clusters.find((c) => c.key === key)
      ),
      [key]
    )
  );
