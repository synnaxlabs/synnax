// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Cluster,
  SLICE_NAME,
  type SliceState,
  type StoreState,
} from "@/cluster/slice";
import { selectByKey, selectByKeys, useMemoSelect } from "@/hooks";

/**
 * Selects the cluster state.
 * @param state - The state of the cluster store.
 */
export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

/** Selects the cluster state. */
export const useSelectSliceState = (): SliceState =>
  useMemoSelect((s: StoreState) => selectSliceState(s), []);

/**
 * Selects the key of the active cluster.
 * @param state - The state of the cluster store.
 */
export const selectActiveKey = (state: StoreState): string | undefined =>
  selectSliceState(state).activeCluster ?? undefined;

/** Selects the key of the active cluster */
export const useSelectActiveKey = (): string | undefined =>
  useMemoSelect((s: StoreState) => selectActiveKey(s), []);

/**
 * Selects a cluster from the cluster store.
 * @param state  - The state of the cluster store.
 * @param key  - The key of the cluster to select. If not provided, the active cluster
 * key will be used. If the active cluster key is not set or the cluster does not exist,
 * null will be returned.
 */
export const select = (state: StoreState, key?: string): Cluster | undefined =>
  selectByKey(selectSliceState(state).clusters, key, selectActiveKey(state));

/**
 * Selects a cluster from the cluster store.
 *
 * @param key  - The key of the cluster to select. If not provided, the active cluster
 * key will be used. If the active cluster key is not set or the cluster does not exist,
 * null will be returned.
 */
export const useSelect = (key?: string): Cluster | undefined =>
  useMemoSelect((s: StoreState) => select(s, key), [key]);

/**
 * Selects a subset of clusters from the cluster store.
 *
 * @param state  - The state of the cluster store.
 * @param keys  - The keys of the clusters to select. If not provided, all clusters are
 * selected.
 */
export const selectMany = (state: StoreState, keys?: string[]): Cluster[] =>
  selectByKeys(state.cluster.clusters, keys);

/**
 * Selects a subset of clusters from the cluster store.
 *
 * @param keys - The keys of the clusters to select. If not provided, all clusters are
 * selected.
 */
export const useSelectMany = (keys?: string[]): Cluster[] =>
  useMemoSelect((s: StoreState) => selectMany(s, keys), [keys]);

export const selectAllNames = (state: StoreState): string[] =>
  Object.values(selectSliceState(state).clusters).map((c) => c.name);

export const useSelectAllNames = (): string[] =>
  useMemoSelect((s: StoreState) => selectAllNames(s), []);
