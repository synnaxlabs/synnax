// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { useCallback } from "react";
import { useStore } from "react-redux";

import {
  type BottomState,
  type LeftState,
  SLICE_NAME,
  type SliceState,
  type StoreState,
  ZERO_WINDOW_STATE,
} from "@/session/nav/slice";
import { Select } from "@/session/select";

interface RequiredStoreState extends StoreState, Drift.StoreState {}

const selectSliceState = (state: RequiredStoreState): SliceState => state[SLICE_NAME];

export const selectWindowState = (state: RequiredStoreState) => {
  const windowKey = Drift.selectWindowKey(state);
  if (windowKey == null) return ZERO_WINDOW_STATE;
  return selectSliceState(state).windows[windowKey] ?? ZERO_WINDOW_STATE;
};

const selectLeft = (state: RequiredStoreState): LeftState =>
  selectWindowState(state).left;

export const useSelectLeft = (): LeftState => Select.useMemo(selectLeft, []);

export const useGetLeft = (): (() => LeftState) => {
  const store = useStore<RequiredStoreState>();
  return useCallback(() => selectLeft(store.getState()), [store]);
};

export const selectLeftSelected = (state: RequiredStoreState): string | undefined =>
  selectLeft(state).selected;

export const useSelectLeftSelected = (): string | undefined =>
  Select.useMemo(selectLeftSelected, []);

export const useGetLeftSelected = (): (() => string | undefined) => {
  const store = useStore<RequiredStoreState>();
  return useCallback(() => selectLeftSelected(store.getState()), [store]);
};

const selectBottom = (state: RequiredStoreState): BottomState =>
  selectWindowState(state).bottom;

export const useSelectBottom = (): BottomState => Select.useMemo(selectBottom, []);

export const useGetBottom = (): (() => BottomState) => {
  const store = useStore<RequiredStoreState>();
  return useCallback(() => selectBottom(store.getState()), [store]);
};

const selectBottomVisible = (state: RequiredStoreState): boolean =>
  selectBottom(state).visible;

export const useSelectBottomVisible = (): boolean =>
  Select.useMemo(selectBottomVisible, []);

export const useGetBottomVisible = (): (() => boolean) => {
  const store = useStore<RequiredStoreState>();
  return useCallback(() => selectBottomVisible(store.getState()), [store]);
};
