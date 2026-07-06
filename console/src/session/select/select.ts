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

export const useMemo = <S extends object, R>(
  selector: (state: S) => R,
  deps: unknown[],
): R => useSelector(useCallback(memoize(selector), deps));

export const byKeys = <K extends record.Key, S extends record.Keyed<K>>(
  state: S[] | Record<K, S>,
  keys?: K[],
): S[] => {
  if (!Array.isArray(state)) state = Object.values(state);
  if (keys == null) return state;
  return state.filter((s) => keys.includes(s.key));
};

export const byKey = <K extends record.Key, S extends record.Keyed<K>>(
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
