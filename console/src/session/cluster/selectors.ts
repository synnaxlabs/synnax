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
} from "@/session/cluster/slice";
import { Select } from "@/session/select";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const useSelectSliceState = (): SliceState =>
  Select.useMemo((s: StoreState) => selectSliceState(s), []);

export const selectActiveKey = (state: StoreState): string | undefined =>
  selectSliceState(state).activeCluster ?? undefined;

export const useSelectActiveKey = (): string | undefined =>
  Select.useMemo((s: StoreState) => selectActiveKey(s), []);

export const select = (state: StoreState, key?: string): Cluster | undefined =>
  Select.byKey(selectSliceState(state).clusters, key, selectActiveKey(state));

export const useSelect = (key?: string): Cluster | undefined =>
  Select.useMemo((s: StoreState) => select(s, key), [key]);

export const selectMany = (state: StoreState, keys?: string[]): Cluster[] =>
  Select.byKeys(state.cluster.clusters, keys);

export const useSelectMany = (keys?: string[]): Cluster[] =>
  Select.useMemo((s: StoreState) => selectMany(s, keys), [keys]);

export const selectAllNames = (state: StoreState): string[] =>
  Object.values(selectSliceState(state).clusters).map((c) => c.name);

export const useSelectAllNames = (): string[] =>
  Select.useMemo((s: StoreState) => selectAllNames(s), []);
