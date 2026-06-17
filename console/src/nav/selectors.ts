// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemoSelect } from "@/hooks";
import {
  type LeftItem,
  SLICE_NAME,
  type SliceState,
  type StoreState,
} from "@/nav/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectLeft = (state: StoreState): SliceState["left"] =>
  selectSliceState(state).left;

export const useSelectLeft = (): SliceState["left"] => useMemoSelect(selectLeft, []);

export const selectLeftSelected = (state: StoreState): LeftItem | undefined =>
  selectLeft(state).selected;

export const useSelectLeftSelected = (): LeftItem | undefined =>
  useMemoSelect(selectLeftSelected, []);

export const selectBottom = (state: StoreState): SliceState["bottom"] =>
  selectSliceState(state).bottom;

export const useSelectBottom = (): SliceState["bottom"] =>
  useMemoSelect(selectBottom, []);

export const selectBottomVisible = (state: StoreState): boolean =>
  selectBottom(state).visible;

export const useSelectBottomVisible = (): boolean =>
  useMemoSelect(selectBottomVisible, []);
