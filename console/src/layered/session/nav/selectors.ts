// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";

import { useMemoSelect } from "@/hooks";
import {
  type BottomState,
  type LeftState,
  SLICE_NAME,
  type SliceState,
  type StoreState,
  ZERO_WINDOW_STATE,
} from "@/layered/session/nav/slice";

interface RequiredStoreState extends StoreState, Drift.StoreState {}

export const selectSliceState = (state: RequiredStoreState): SliceState =>
  state[SLICE_NAME];

export const selectWindowState = (state: RequiredStoreState) => {
  const windowKey = Drift.selectWindowKey(state);
  if (windowKey == null) return ZERO_WINDOW_STATE;
  return selectSliceState(state).windows[windowKey] ?? ZERO_WINDOW_STATE;
};

const selectLeft = (state: RequiredStoreState): LeftState =>
  selectWindowState(state).left;

export const useSelectLeft = (): LeftState => useMemoSelect(selectLeft, []);

export const selectLeftSelected = (state: RequiredStoreState): string | undefined =>
  selectLeft(state).selected;

export const useSelectLeftSelected = (): string | undefined =>
  useMemoSelect(selectLeftSelected, []);

const selectBottom = (state: RequiredStoreState): BottomState =>
  selectWindowState(state).bottom;

export const useSelectBottom = (): BottomState => useMemoSelect(selectBottom, []);

export const selectBottomVisible = (state: RequiredStoreState): boolean =>
  selectBottom(state).visible;

export const useSelectBottomVisible = (): boolean =>
  useMemoSelect(selectBottomVisible, []);
