// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { Ranger } from "@synnaxlabs/pluto";
import { type NumericTimeRange } from "@synnaxlabs/x";
import { useMemo } from "react";

import { Session } from "@/session";

/** A Core range with the name and time range the Core answers with folded in. */
export interface ResolvedPersisted extends Session.Range.PersistedState {
  name: string;
  timeRange: NumericTimeRange;
}

/**
 * A session range every consumer can render. `variant` still says where the range
 * lives, so a caller that only cares about the Core's own ranges can still tell.
 */
export type Resolved =
  ResolvedPersisted | Session.Range.StaticState | Session.Range.DynamicState;

const fold = (
  state: Session.Range.State,
  found: Map<string, ranger.Range>,
): Resolved | undefined => {
  if (state.variant !== "persisted") return state;
  const range = found.get(state.key);
  // A range the Core no longer has resolves to nothing until the synchronizer drops it,
  // which keeps a half-rendered row out of the list.
  if (range == null) return undefined;
  return { ...state, name: range.name, timeRange: range.timeRange.numeric };
};

/**
 * Resolves the session's ranges, reading the Core for the ones it holds.
 * @param keys - The ranges to resolve. Defaults to every one the session has.
 */
export const useResolveMultiple = (keys?: string[]): Resolved[] => {
  const entries = Session.Range.useSelectMultiple(keys);
  const persisted = useMemo(
    () =>
      entries.filter(({ variant }) => variant === "persisted").map(({ key }) => key),
    [entries],
  );
  const { data } = Ranger.useResultMultiple(
    persisted.length === 0 ? null : { keys: persisted },
  );
  return useMemo(() => {
    const found = new Map((data ?? []).map((range) => [range.key, range]));
    return entries.map((entry) => fold(entry, found)).filter((entry) => entry != null);
  }, [entries, data]);
};

/**
 * Resolves one of the session's ranges.
 * @param key - The range to resolve. Defaults to the selected one.
 */
export const useResolve = (key?: string): Resolved | undefined => {
  const entry = Session.Range.useSelectState(key);
  const { data } = Ranger.useResult(
    entry?.variant === "persisted" ? { key: entry.key } : null,
  );
  return useMemo(() => {
    if (entry == null) return undefined;
    const found = new Map(data == null ? [] : [[data.key, data] as const]);
    return fold(entry, found);
  }, [entry, data]);
};
