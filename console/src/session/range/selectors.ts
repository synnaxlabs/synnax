// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  BUILT_IN,
  type Group,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
} from "@/session/range/slice";
import { Select } from "@/session/select";

const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

// The built-ins lead so they hold a stable spot at the top of every list.
const selectAll = (state: StoreState): State[] => [
  ...BUILT_IN,
  ...selectSliceState(state).ranges,
];

const selectState = (state: StoreState, key: string): State | undefined =>
  selectAll(state).find((r) => r.key === key);

export const useSelectState = (key: string): State | undefined =>
  Select.useMemo((state: StoreState) => selectState(state, key), [key]);

const selectMultiple = (state: StoreState, keys?: string[]): State[] => {
  const all = selectAll(state);
  if (keys == null) return all;
  return all.filter((range) => keys.includes(range.key));
};

export const useSelectMultiple = (keys?: string[]): State[] =>
  Select.useMemo((state: StoreState) => selectMultiple(state, keys), [keys]);

const selectFavoriteKeys = (state: StoreState): string[] =>
  selectSliceState(state).ranges.map((r) => r.key);

/** @returns the keys of the Core ranges the session added. */
export const useSelectFavoriteKeys = (): string[] =>
  Select.useMemo((state: StoreState) => selectFavoriteKeys(state), []);

const selectGroupCollapsed = (state: StoreState, group: Group): boolean =>
  selectSliceState(state).toolbar.collapsed.includes(group);

export const useSelectGroupCollapsed = (group: Group): boolean =>
  Select.useMemo((state: StoreState) => selectGroupCollapsed(state, group), [group]);
