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
  type PendingUpload,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
} from "@/table/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).tables[key];

export const selectOptional = select as (
  state: StoreState,
  key: string,
) => State | undefined;

export const useSelect = (key: string): State =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const useSelectOptional = (key: string): State | undefined =>
  useMemoSelect((state: StoreState) => selectOptional(state, key), [key]);

export const selectEditable = (state: StoreState, key: string): boolean =>
  selectOptional(state, key)?.editable ?? false;

export const useSelectEditable = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectEditable(state, key), [key]);

export const selectHideIndicators = (state: StoreState, key: string): boolean =>
  selectOptional(state, key)?.hideIndicators ?? false;

export const useSelectHideIndicators = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectHideIndicators(state, key), [key]);

export const selectSelectedCellKeys = (state: StoreState, key: string): string[] =>
  selectOptional(state, key)?.selectedCells ?? [];

export const useSelectSelectedCellKeys = (key: string): string[] =>
  useMemoSelect((state: StoreState) => selectSelectedCellKeys(state, key), [key]);

export const selectLastSelected = (state: StoreState, key: string): string | null =>
  selectOptional(state, key)?.lastSelected ?? null;

export const useSelectLastSelected = (key: string): string | null =>
  useMemoSelect((state: StoreState) => selectLastSelected(state, key), [key]);

export const selectVersion = (state: StoreState, key: string): string | undefined =>
  selectOptional(state, key)?.version;

export const useSelectVersion = (key: string): string | undefined =>
  useMemoSelect((state: StoreState) => selectVersion(state, key), [key]);

export const selectPendingUpload = (
  state: StoreState,
  key: string,
): PendingUpload | undefined => selectOptional(state, key)?.pendingUpload;

export const useSelectPendingUpload = (key: string): PendingUpload | undefined =>
  useMemoSelect((state: StoreState) => selectPendingUpload(state, key), [key]);
