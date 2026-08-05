// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";

import {
  type BottomState,
  type LeftState,
  SLICE_NAME,
  type SliceState,
  type StoreState,
  ZERO_WINDOW_STATE,
} from "@/session/nav/slice";
import { Select } from "@/session/select";

const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectWindowState = (state: StoreState) => {
  const windowKey = Drift.selectWindowKey(state);
  if (windowKey == null) return ZERO_WINDOW_STATE;
  return selectSliceState(state).windows[windowKey] ?? ZERO_WINDOW_STATE;
};

const selectLeft = (state: StoreState): LeftState => selectWindowState(state).left;

export const useSelectLeft = (): LeftState => Select.useMemo(selectLeft, []);

export const selectLeftSelected = (state: StoreState): string | undefined =>
  selectLeft(state).selected;

export const useSelectLeftSelected = (): string | undefined =>
  Select.useMemo(selectLeftSelected, []);

const selectBottom = (state: StoreState): BottomState =>
  selectWindowState(state).bottom;

export const useSelectBottom = (): BottomState => Select.useMemo(selectBottom, []);

const selectBottomVisible = (state: StoreState): boolean => selectBottom(state).visible;

export const useSelectBottomVisible = (): boolean =>
  Select.useMemo(selectBottomVisible, []);
