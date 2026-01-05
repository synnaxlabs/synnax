// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { selectByKey, useMemoSelect } from "@/hooks";
import {
  type Range,
  SLICE_NAME,
  type SliceState,
  type StoreState,
} from "@/range/slice";
import { type StaticRange } from "@/range/types";

/**
 * Selects the workspace state.
 *
 * @param state - The state of the workspace store.
 * @returns The workspace state.
 */
const selectState = (state: StoreState): SliceState => state[SLICE_NAME];

/**
 * Selects the workspace state.
 *
 * @returns The workspace state.
 */
export const useSelectState = (): SliceState =>
  useMemoSelect((state: StoreState) => selectState(state), []);

/**
 * Selects the key of the active range.
 *
 * @param state - The state of the workspace store.
 * @returns The key of the active range, or null if no range is active.
 */
export const selectActiveKey = (state: StoreState): string | null =>
  selectState(state).activeRange;

/**
 * Selects the key of the active range.
 *
 * @returns The key of the active range, or null if no range is active.
 */
export const useSelectActiveKey = (): string | null =>
  useMemoSelect((state: StoreState) => selectActiveKey(state), []);

/**
 * Selects a range from the workspace store.
 *
 * @param state - The state of the workspace store.
 * @param key - The key of the range to select. If not provided, the active range key
 * will be used.
 *
 * @returns The range with the given key. If the range does not exist, returns
 * undefined. If no key is provided, the active range key is used. If no active range is
 * set, returns null.
 */
export const select = (state: StoreState, key?: string): Range | undefined =>
  selectByKey(selectState(state).ranges, key, selectActiveKey(state));

/**
 * Selects a range from the workspace store.
 *
 * @param key - The key of the range to select. If not provided, the active range key
 * will be used.
 *
 * @returns The range with the given key. If the range does not exist, returns
 * undefined. If no key is provided, the active range key is used. If no active range is
 * set, returns null.
 */
export const useSelect = (key?: string): Range | undefined =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const selectStatic = (
  state: StoreState,
  key?: string,
): StaticRange | undefined => {
  const range = select(state, key);
  if (range == null) return undefined;
  if (range.variant !== "static") return undefined;
  return range;
};

export const useSelectStatic = (key?: string): StaticRange | undefined =>
  useMemoSelect((state: StoreState) => selectStatic(state, key), [key]);

/**
 * Selects ranges with the given keys. If no keys are provided, all ranges are selected.
 *
 * @param state  - The state of the workspace store.
 * @param keys  - The keys of the ranges to select. If not provided, all ranges are
 * selected.
 * @returns The ranges with the given keys.
 */
export const selectMultiple = (state: StoreState, keys?: string[]): Range[] => {
  const all = Object.values(selectState(state).ranges);
  if (keys == null) return all;
  return all.filter((range) => keys.includes(range.key));
};

/**
 * Selects ranges from the workspace store. If no keys are provided, all ranges are
 * selected.
 *
 * @param keys - The keys of the ranges to select. If not provided, all ranges are
 * selected.
 * @returns The ranges with the given keys.
 */
export const useSelectMultiple = (keys?: string[]): Range[] =>
  useMemoSelect((state: StoreState) => selectMultiple(state, keys), [keys]);

export const selectKeys = (state: StoreState): string[] =>
  Object.keys(selectState(state).ranges);

export const useSelectKeys = (): string[] =>
  useMemoSelect((state: StoreState) => selectKeys(state), []);

export const selectStaticKeys = (state: StoreState): string[] =>
  Object.keys(selectState(state).ranges).filter((key) => {
    const range = select(state, key);
    return range != null && range.variant === "static";
  });

export const useSelectStaticKeys = (): string[] =>
  useMemoSelect((state: StoreState) => selectStaticKeys(state), []);
