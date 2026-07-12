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
  SLICE_NAME,
  type SliceState,
  type State,
  type StaticState,
  type StoreState,
} from "@/session/range/slice";
import { Select } from "@/session/select";

const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

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
  const { ranges } = selectSliceState(state);
  key ??= selectSelectedKey(state);
  return ranges.find((r) => r.key === key);
};

export const useSelectState = (key?: string): State | undefined =>
  Select.useMemo((state: StoreState) => selectState(state, key), [key]);

export const useGetState = (): ((key?: string) => State | undefined) => {
  const store = useStore<StoreState>();
  return useCallback((key?: string) => selectState(store.getState(), key), [store]);
};

const selectStatic = (state: StoreState, key?: string): StaticState | undefined => {
  const range = selectState(state, key);
  if (range?.variant !== "static") return undefined;
  return range;
};

export const useSelectStatic = (key?: string): StaticState | undefined =>
  Select.useMemo((state: StoreState) => selectStatic(state, key), [key]);

export const useGetStatic = (): ((key?: string) => StaticState | undefined) => {
  const store = useStore<StoreState>();
  return useCallback((key?: string) => selectStatic(store.getState(), key), [store]);
};

export const selectMultiple = (state: StoreState, keys?: string[]): State[] => {
  const { ranges } = selectSliceState(state);
  if (keys == null) return ranges;
  return ranges.filter((range) => keys.includes(range.key));
};

export const useSelectMultiple = (keys?: string[]): State[] =>
  Select.useMemo((state: StoreState) => selectMultiple(state, keys), [keys]);

export const useGetMultiple = (): ((keys?: string[]) => State[]) => {
  const store = useStore<StoreState>();
  return useCallback(
    (keys?: string[]) => selectMultiple(store.getState(), keys),
    [store],
  );
};

export const selectKeys = (state: StoreState): string[] =>
  selectSliceState(state).ranges.map((r) => r.key);

export const useSelectKeys = (): string[] =>
  Select.useMemo((state: StoreState) => selectKeys(state), []);

export const useGetKeys = (): (() => string[]) => {
  const store = useStore<StoreState>();
  return useCallback(() => selectKeys(store.getState()), [store]);
};

const selectStaticKeys = (state: StoreState): string[] =>
  selectSliceState(state)
    .ranges.filter((r) => r.variant === "static")
    .map((r) => r.key);

export const useSelectStaticKeys = (): string[] =>
  Select.useMemo((state: StoreState) => selectStaticKeys(state), []);

export const useGetStaticKeys = (): (() => string[]) => {
  const store = useStore<StoreState>();
  return useCallback(() => selectStaticKeys(store.getState()), [store]);
};
