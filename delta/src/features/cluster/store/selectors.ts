import { Cluster } from "../types";

import { ClusterStoreState } from "./slice";

import { selectByKey, selectByKeys, useMemoSelect } from "@/hooks";

/**
 * Selects the key of the active cluster.
 * @param state - The state of the cluster store.
 */
export const selectActiveClusterKey = (state: ClusterStoreState): string | null =>
  state.cluster.activeCluster;

/** Selects the key of the active cluster */
export const useSelectActiveClusterKey = (): string | null =>
  useMemoSelect(selectActiveClusterKey, []);

/**
 * Selects a cluster from the cluster store.
 * @param state  - The state of the cluster store.
 * @param key  - The key of the cluster to select. If not provided, the active cluster
 * key will be used. If the active cluster key is not set or the cluster does not exist,
 * null will be returned.
 */
export const selectCluster = (
  state: ClusterStoreState,
  key?: string | null
): Cluster | null | undefined =>
  selectByKey(state.cluster.clusters, key, selectActiveClusterKey(state));

/**
 * Selects a cluster from the cluster store.
 *
 * @param state  - The state of the cluster store.
 * @param key  - The key of the cluster to select. If not provided, the active cluster
 * key will be used. If the active cluster key is not set or the cluster does not exist,
 * null will be returned.
 */
export const useSelectCluster = (key?: string): Cluster | null | undefined =>
  useMemoSelect((s: ClusterStoreState) => selectCluster(s, key), [key]);

/**
 * Selects a subset of clusters from the cluster store.
 *
 * @param s  - The state of the cluster store.
 * @param keys  - The keys of the clusters to select. If not provided, all clusters are
 * selected.
 */
export const selectClusters = (s: ClusterStoreState, keys?: string[]): Cluster[] =>
  selectByKeys(s.cluster.clusters, keys);

/**
 * Selects a subset of clusters from the cluster store.
 *
 * @param keys - The keys of the clusters to select. If not provided, all clusters are
 * selected.
 */
export const useSelectClusters = (keys?: string[]): Cluster[] =>
  useMemoSelect((s: ClusterStoreState) => selectClusters(s, keys), [keys]);
