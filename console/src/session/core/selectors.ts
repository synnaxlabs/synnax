// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";
import { useStore } from "react-redux";

import {
  type Core,
  SLICE_NAME,
  type SliceState,
  type StoreState,
} from "@/session/core/slice";
import { Select } from "@/session/select";

const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const useSelectSliceState = (): SliceState =>
  Select.useMemo((s: StoreState) => selectSliceState(s), []);

export const useGetSliceState = (): (() => SliceState) => {
  const store = useStore<StoreState>();
  return useCallback(() => selectSliceState(store.getState()), [store]);
};

export const selectSelectedKey = (state: StoreState): string | undefined =>
  selectSliceState(state).selected;

export const useSelectSelectedKey = (): string | undefined =>
  Select.useMemo((s: StoreState) => selectSelectedKey(s), []);

export const useGetSelectedKey = (): (() => string | undefined) => {
  const store = useStore<StoreState>();
  return useCallback(() => selectSelectedKey(store.getState()), [store]);
};

/** The Core stored under key, or nothing when key names none. */
export const selectState = (state: StoreState, key?: string): Core | undefined =>
  Select.byKey(selectSliceState(state).cores, key);

export const useSelectState = (key?: string): Core | undefined =>
  Select.useMemo((s: StoreState) => selectState(s, key), [key]);

export const useGetState = (): ((key?: string) => Core | undefined) => {
  const store = useStore<StoreState>();
  return useCallback((key?: string) => selectState(store.getState(), key), [store]);
};

export const selectSelected = (state: StoreState): Core | undefined =>
  selectState(state, selectSelectedKey(state));

export const useSelectSelected = (): Core | undefined =>
  Select.useMemo(selectSelected, []);

/** The cluster the selected Core last connected to. */
export const selectClusterKey = (state: StoreState): string | undefined =>
  selectSelected(state)?.clusterKey;

export const useSelectClusterKey = (): string | undefined =>
  Select.useMemo(selectClusterKey, []);

/**
 * The Core to reach the given cluster through, preferring the selected one so a link to
 * the cluster already open does not switch Cores.
 */
export const selectByClusterKey = (
  state: StoreState,
  clusterKey: string,
): Core | undefined => {
  const selected = selectSelected(state);
  if (selected?.clusterKey === clusterKey) return selected;
  return Object.values(selectSliceState(state).cores).find(
    (c) => c.clusterKey === clusterKey,
  );
};

/**
 * Whether no Core still names the cluster. Its stored state is unreachable once true.
 * Ask after the change that dropped the name, not before.
 */
export const selectIsClusterOrphaned = (
  state: StoreState,
  clusterKey: string,
): boolean =>
  !Object.values(selectSliceState(state).cores).some(
    (c) => c.clusterKey === clusterKey,
  );

export const selectMany = (state: StoreState, keys?: string[]): Core[] =>
  Select.byKeys(state.core.cores, keys);

export const useSelectMany = (keys?: string[]): Core[] =>
  Select.useMemo((s: StoreState) => selectMany(s, keys), [keys]);

const selectIsAnySelected = (state: StoreState): boolean =>
  selectSelectedKey(state) != null;

export const useSelectIsAnySelected = (): boolean =>
  Select.useMemo(selectIsAnySelected, []);

export const useGetIsAnySelected = (): (() => boolean) => {
  const store = useStore<StoreState>();
  return useCallback(() => selectIsAnySelected(store.getState()), [store]);
};
