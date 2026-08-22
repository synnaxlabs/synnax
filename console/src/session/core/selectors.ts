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

export const selectState = (state: StoreState, key?: string): Core | undefined =>
  Select.byKey(selectSliceState(state).cores, key, selectSelectedKey(state));

export const useSelectState = (key?: string): Core | undefined =>
  Select.useMemo((s: StoreState) => selectState(s, key), [key]);

export const useGetState = (): ((key?: string) => Core | undefined) => {
  const store = useStore<StoreState>();
  return useCallback((key?: string) => selectState(store.getState(), key), [store]);
};

export const selectMany = (state: StoreState, keys?: string[]): Core[] =>
  Select.byKeys(state.core.cores, keys);

export const useSelectMany = (keys?: string[]): Core[] =>
  Select.useMemo((s: StoreState) => selectMany(s, keys), [keys]);

const selectAllNames = (state: StoreState): string[] =>
  Object.values(selectSliceState(state).cores).map((c) => c.name);

export const useSelectAllNames = (): string[] =>
  Select.useMemo((s: StoreState) => selectAllNames(s), []);

const selectIsAnySelected = (state: StoreState): boolean =>
  selectSelectedKey(state) != null;

export const useSelectIsAnySelected = (): boolean =>
  Select.useMemo(selectIsAnySelected, []);

export const useGetIsAnySelected = (): (() => boolean) => {
  const store = useStore<StoreState>();
  return useCallback(() => selectIsAnySelected(store.getState()), [store]);
};
