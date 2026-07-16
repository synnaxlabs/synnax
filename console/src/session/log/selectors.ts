// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type log } from "@synnaxlabs/client";
import { Log } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

import {
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarTab,
  ZERO_STATE,
} from "@/session/log/slice";
import { Select } from "@/session/select";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export interface KeyedSelectorParams extends record.Keyed<log.Key> {
  state: StoreState;
}

const createSelector = <R>(selector: (params: KeyedSelectorParams) => R) =>
  Log.Scope.bindHook(({ key }: Omit<KeyedSelectorParams, "state">): R =>
    Select.useMemo((state: StoreState) => selector({ state, key }), [key]),
  );

export const selectState = ({ state, key }: KeyedSelectorParams): State =>
  selectSliceState(state).logs[key] ?? ZERO_STATE;

export const useSelectState = createSelector(selectState);

export const selectSelectedToolbarTab = (params: KeyedSelectorParams): ToolbarTab =>
  selectState(params).toolbar.selectedTab;

export const useSelectSelectedToolbarTab = createSelector(selectSelectedToolbarTab);
