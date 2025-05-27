// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Effect,
  SLICE_NAME,
  type SliceState,
  type StoreState,
} from "@/effect/slice";
import { selectByKey, useMemoSelect } from "@/hooks";

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
export const select = (
  state: StoreState,
  key?: string | null,
): Effect | null | undefined => selectByKey(selectState(state).effects, key);

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
export const useSelect = (key: string): Effect | null | undefined =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

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
): Effect[] => {
  const all = Object.values(selectState(state).effects);
  if (keys == null) return all;
  return all.filter((effect) => keys.includes(effect.key));
};

/**
 * Selects ranges from the workspace store. If no keys are provided, all ranges are
 * selected.
 *
 * @param keys - The keys of the ranges to select. If not provided, all ranges are
 * selected.
 * @returns The ranges with the given keys.
 */
export const useSelectMultiple = (keys?: string[]): Effect[] =>
  useMemoSelect((state: StoreState) => selectMultiple(state, keys), [keys]);

export const selectKeys = (state: StoreState): string[] =>
  Object.keys(selectState(state).effects);

export const useSelectKeys = (): string[] =>
  useMemoSelect((state: StoreState) => selectKeys(state), []);
