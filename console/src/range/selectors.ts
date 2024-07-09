// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { selectByKey, useMemoSelect } from "@/hooks";
import type { Range, StaticRange } from "@/range/slice";
import { SLICE_NAME, type SliceState, type StoreState } from "@/range/slice";

/**
 * Selects the workspace state.
 * @param state - The state of the workspace store.
 * @returns The workspace state.
 */
const selectState = (state: StoreState): SliceState => state[SLICE_NAME];

/**
 * Selects ranges with the given keys. If no keys are provided, all ranges are selected.
 *
 * @param state  - The state of the workspace store.
 * @param keys  - The keys of the ranges to select. If not provided, all ranges are
 * selected.
 * @returns The ranges with the given keys.
 */
export const selectMultiple = (
  state: StoreState,
  keys?: string[] | string[],
): Range[] => {
  const all = Object.values(selectState(state).ranges);
  if (keys == null) return all;
  return all.filter((range) => keys.includes(range.key));
};
/**
 * Selects the key of the active range.
 *
 * @param state - The state of the workspace store.
 * @returns The key of the active range, or null if no range is active.
 */
export const selectActiveKey = (state: StoreState): string | null =>
  selectState(state).activeRange;

/**
 * Selects a range from the workspace store.
 *
 * @param state - The state of the workspace store.
 * @param key - The key of the range to select. If not provided, the active range key
 * will be used.
 *
 * @returns The range with the given key. If no key is provided, the active range is
 * key is used. If no active range is set, returns null. If the range does not exist,
 * returns undefined.
 */
export const select = (
  state: StoreState,
  key?: string | null,
): Range | null | undefined =>
  selectByKey(selectState(state).ranges, key, selectActiveKey(state));

/**
 * Selects information from the current edit range buffer.
 *
 * @param state - The state of the workspace store.
 *
 * @returns Information from the stored buffer. If no buffer is set, it returns null.
 */
export const selectBuffer = (
  state: StoreState,
): Partial<StaticRange> | null | undefined => selectState(state).buffer;

/**
 * Selects a range from the workspace store.
 *
 * @returns The range with the given key. If no key is provided, the active range is
 * key is used. If no active range is set, returns null. If the range does not exist,
 * returns undefined.
 */
export const useSelect = (key?: string): Range | null | undefined =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

/**
 * Selects information from the current edit range buffer.
 *
 * @returns Information from the stored buffer. If no buffer is set, it returns null.
 */
export const useSelectBuffer = (): Partial<Range> | null | undefined =>
  useMemoSelect((state: StoreState) => selectBuffer(state), []);

/**
 * Selects ranges from the workspace store. If no keys are provided, all ranges are
 * selected.
 *
 * @param keys - The keys of the ranges to select. If not provided, all ranges are
 * @returns The ranges with the given keys.
 */
export const useSelectMultiple = (keys?: string[]): Range[] =>
  useMemoSelect((state: StoreState) => selectMultiple(state, keys), [keys]);
