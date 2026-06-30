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
  type Range,
  SLICE_NAME,
  type SliceState,
  type Static,
  type StoreState,
} from "@/layered/session/range/slice";

const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const useSelectSliceState = (): SliceState =>
  useMemoSelect((state: StoreState) => selectSliceState(state), []);

export const selectSelectedKey = (state: StoreState): string | undefined =>
  selectSliceState(state).selected;

export const useSelectSelectedKey = (): string | undefined =>
  useMemoSelect((state: StoreState) => selectSelectedKey(state), []);

const select = (state: StoreState, key?: string): Range | undefined => {
  const { ranges } = selectSliceState(state);
  key ??= selectSelectedKey(state);
  return ranges.find((r) => r.key === key);
};

export const useSelect = (key?: string): Range | undefined =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const selectStatic = (state: StoreState, key?: string): Static | undefined => {
  const range = select(state, key);
  if (range?.variant !== "static") return undefined;
  return range;
};

export const useSelectStatic = (key?: string): Static | undefined =>
  useMemoSelect((state: StoreState) => selectStatic(state, key), [key]);

export const selectMultiple = (state: StoreState, keys?: string[]): Range[] => {
  const { ranges } = selectSliceState(state);
  if (keys == null) return ranges;
  return ranges.filter((range) => keys.includes(range.key));
};

export const useSelectMultiple = (keys?: string[]): Range[] =>
  useMemoSelect((state: StoreState) => selectMultiple(state, keys), [keys]);

export const selectKeys = (state: StoreState): string[] =>
  selectSliceState(state).ranges.map((r) => r.key);

export const useSelectKeys = (): string[] =>
  useMemoSelect((state: StoreState) => selectKeys(state), []);

export const selectStaticKeys = (state: StoreState): string[] =>
  selectSliceState(state)
    .ranges.filter((r) => r.variant === "static")
    .map((r) => r.key);

export const useSelectStaticKeys = (): string[] =>
  useMemoSelect((state: StoreState) => selectStaticKeys(state), []);
