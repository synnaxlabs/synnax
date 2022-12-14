// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Cluster } from "../types";

import { ClusterStoreState } from "./slice";

import { useMemoSelect } from "@/hooks";

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
): Cluster | null => {
  if (key == null) key = selectActiveClusterKey(state);
  if (key == null) return null;
  return state.cluster.clusters[key];
};

/**
 * Selects a cluster from the cluster store.
 * @param state  - The state of the cluster store.
 * @param key  - The key of the cluster to select. If not provided, the active cluster
 * key will be used. If the active cluster key is not set or the cluster does not exist,
 * null will be returned.
 */
export const useSelectCluster = (key?: string): Cluster | null =>
  useMemoSelect((state: ClusterStoreState) => selectCluster(state, key), [key]);

/** Selects all clusters from the cluster store */
export const selectClusters = (state: ClusterStoreState): Cluster[] =>
  Object.values(state.cluster.clusters);

/** Selects all clusters from the cluster store */
export const useSelectClusters = (): Cluster[] => useMemoSelect(selectClusters, []);
