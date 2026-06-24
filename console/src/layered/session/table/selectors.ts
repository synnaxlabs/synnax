// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type table } from "@synnaxlabs/client";
import { Table } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

import { useMemoSelect } from "@/hooks";
import {
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  ZERO_STATE,
} from "@/layered/session/table/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export interface KeyedSelectorParams extends record.Keyed<table.Key> {
  state: StoreState;
}

const createSelector = <R>(selector: (params: KeyedSelectorParams) => R) =>
  Table.Scope.bindHook(
    ({ key }: Omit<KeyedSelectorParams, "state">): R =>
      useMemoSelect((state: StoreState) => selector({ state, key }), [key]),
  );

export const selectState = ({ state, key }: KeyedSelectorParams): State =>
  selectSliceState(state).tables[key] ?? ZERO_STATE;

export const useSelect = createSelector(selectState);

export const selectOptional = ({
  state,
  key,
}: KeyedSelectorParams): State | undefined => selectSliceState(state).tables[key];

export const useSelectOptional = createSelector(selectOptional);

export const selectExists = (params: KeyedSelectorParams): boolean =>
  selectOptional(params) != null;

export const useSelectExists = createSelector(selectExists);

export const selectEditable = (params: KeyedSelectorParams): boolean =>
  selectState(params).editable;

export const useSelectEditable = createSelector(selectEditable);

export const selectHideIndicators = (params: KeyedSelectorParams): boolean =>
  selectState(params).hideIndicators;

export const useSelectHideIndicators = createSelector(selectHideIndicators);

export const selectSelectedCellKeys = (params: KeyedSelectorParams): string[] =>
  selectState(params).selectedCells;

export const useSelectSelectedCellKeys = createSelector(selectSelectedCellKeys);

export const selectLastSelected = (params: KeyedSelectorParams): string | null =>
  selectState(params).lastSelected;

export const useSelectLastSelected = createSelector(selectLastSelected);
