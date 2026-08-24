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
  BUILT_IN,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
} from "@/session/range/slice";
import { Select } from "@/session/select";

const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

// The built-ins lead, as they did when stored state carried them.
const selectAll = (state: StoreState): State[] => [
  ...BUILT_IN,
  ...selectSliceState(state).ranges,
];

export const useSelectSliceState = (): SliceState =>
  Select.useMemo((state: StoreState) => selectSliceState(state), []);

export const useGetSliceState = (): (() => SliceState) => {
  const store = useStore<StoreState>();
  return useCallback(() => selectSliceState(store.getState()), [store]);
};

export const selectSelectedKey = (state: StoreState): string | undefined =>
  selectSliceState(state).selected;

export const useSelectSelectedKey = (): string | undefined =>
  Select.useMemo((state: StoreState) => selectSelectedKey(state), []);

export const useGetSelectedKey = (): (() => string | undefined) => {
  const store = useStore<StoreState>();
  return useCallback(() => selectSelectedKey(store.getState()), [store]);
};

export const selectState = (state: StoreState, key?: string): State | undefined => {
  key ??= selectSelectedKey(state);
  return selectAll(state).find((r) => r.key === key);
};

export const useSelectState = (key?: string): State | undefined =>
  Select.useMemo((state: StoreState) => selectState(state, key), [key]);

export const useGetState = (): ((key?: string) => State | undefined) => {
  const store = useStore<StoreState>();
  return useCallback((key?: string) => selectState(store.getState(), key), [store]);
};

export const selectMultiple = (state: StoreState, keys?: string[]): State[] => {
  const all = selectAll(state);
  if (keys == null) return all;
  return all.filter((range) => keys.includes(range.key));
};

export const useSelectMultiple = (keys?: string[]): State[] =>
  Select.useMemo((state: StoreState) => selectMultiple(state, keys), [keys]);

export const selectKeys = (state: StoreState): string[] =>
  selectAll(state).map((r) => r.key);

export const useSelectKeys = (): string[] =>
  Select.useMemo((state: StoreState) => selectKeys(state), []);

// Ranges covering a fixed window, which is every one the Core could hold.
const selectStaticKeys = (state: StoreState): string[] =>
  selectAll(state)
    .filter((r) => r.variant !== "dynamic")
    .map((r) => r.key);

export const useSelectStaticKeys = (): string[] =>
  Select.useMemo((state: StoreState) => selectStaticKeys(state), []);
