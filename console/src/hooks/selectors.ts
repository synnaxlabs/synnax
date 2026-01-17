// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { memoize } from "proxy-memoize";
import { useCallback } from "react";
import { useSelector } from "react-redux";

/**
 * A memoized version of the redux useSelector hook. Only re-renders when the portions
 * of state accessed by the selector OR its dependencies change.
 *
 * @param selector - The selector function. NOTE: Avoid using object destructuring in
 * the selector, as it may cause issues with memoization.
 * @param deps - The dependencies of the selector. If not provided, the selector will
 * only re-run when the state changes.
 * @returns The result of the selector.
 */
export const useMemoSelect = <S extends object, R>(
  selector: (state: S) => R,
  deps: unknown[],
): R => useSelector(useCallback(memoize(selector), deps));

export const selectByKeys = <K extends record.Key, S extends record.Keyed<K>>(
  state: S[] | Record<K, S>,
  keys?: K[],
): S[] => {
  if (!Array.isArray(state)) state = Object.values(state);
  if (keys == null) return state;
  return state.filter((s) => keys.includes(s.key));
};

export const selectByKey = <K extends record.Key, S extends record.Keyed<K>>(
  state: Record<string, S>,
  key?: string | null,
  defaultKey?: string | null,
): S | undefined => {
  key ??= defaultKey;
  if (key == null) return undefined;
  const res = state[key];
  if (res != null) return res;
  return Object.values(state).find((s) => s.key === key);
};
