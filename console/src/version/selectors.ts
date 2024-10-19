// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemoSelect } from "@/hooks";
import { SLICE_NAME, type SliceState, type StoreState } from "@/version/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const useSelectSliceState = (): SliceState =>
  useMemoSelect((state: StoreState) => selectSliceState(state), []);

export const selectVersion = (state: StoreState): string =>
  selectSliceState(state).consoleVersion;

export const useSelectVersion = (): string => useMemoSelect(selectVersion, []);

export const selectSilenced = (state: StoreState): boolean =>
  selectSliceState(state).silenced;

export const useSelectSilenced = (): boolean => useMemoSelect(selectSilenced, []);
