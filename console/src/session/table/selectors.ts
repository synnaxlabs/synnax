// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type table } from "@synnaxlabs/client";
import { type Properties, Table } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

import { Select } from "@/session/select";
import {
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  ZERO_STATE,
} from "@/session/table/slice";
import { Window } from "@/session/window";

const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export interface KeyedSelectorParams extends record.Keyed<table.Key> {
  state: StoreState;
}

const createSelector = <R>(selector: (params: KeyedSelectorParams) => R) => {
  const useSelector = ({ key }: Omit<KeyedSelectorParams, "state">): R =>
    Select.useMemo((state: StoreState) => selector({ state, key }), [key]);
  return Table.Scope.bindHook(useSelector);
};

const selectState = ({ state, key }: KeyedSelectorParams): State =>
  Window.selectDocument(state, selectSliceState(state), key) ?? ZERO_STATE;

const selectEditable = (params: KeyedSelectorParams): boolean =>
  selectState(params).editable;

export const useSelectEditable = createSelector(selectEditable);

const selectHideIndicators = (params: KeyedSelectorParams): boolean =>
  selectState(params).hideIndicators;

export const useSelectHideIndicators = createSelector(selectHideIndicators);

const selectCentered = (params: KeyedSelectorParams): boolean =>
  selectState(params).centered;

export const useSelectCentered = createSelector(selectCentered);

const selectPropertiesTab = (
  params: KeyedSelectorParams,
): Properties.TabKey | undefined => selectState(params).propertiesTab;

export const useSelectPropertiesTab = createSelector(selectPropertiesTab);

const selectSelectedCellKeys = (params: KeyedSelectorParams): string[] =>
  selectState(params).selectedCells;

export const useSelectSelectedCellKeys = createSelector(selectSelectedCellKeys);
