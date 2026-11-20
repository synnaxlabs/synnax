// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ranger } from "@synnaxlabs/client";
import { Ranger } from "@synnaxlabs/pluto";
import { dynamicWindow, staticWindow, type Window } from "@synnaxlabs/x";

import { BUILT_IN, type State } from "@/session/range/slice";

/** @returns the reference a visualization stores to show the session range. */
export const toRef = (state: State): ranger.Ref => {
  switch (state.variant) {
    case "persisted":
      return { variant: "key", key: state.key };
    case "dynamic":
      return { variant: "window", window: dynamicWindow(state.span) };
  }
};

/** The reference that new plots and channels show: the shortest rolling window. */
export const RECENT_REF: ranger.Ref = toRef(BUILT_IN[0]);

const windowsEqual = (a: Window, b: Window): boolean => {
  if (a.variant === "dynamic")
    return b.variant === "dynamic" && a.timeSpan === b.timeSpan;
  return (
    b.variant === "static" &&
    a.timeRange.start === b.timeRange.start &&
    a.timeRange.end === b.timeRange.end
  );
};

/** @returns whether the two references show the same range. */
export const refsEqual = (a: ranger.Ref, b: ranger.Ref): boolean => {
  if (a.variant === "key") return b.variant === "key" && a.key === b.key;
  return b.variant === "window" && windowsEqual(a.window, b.window);
};

/**
 * @returns the key of the first session range the reference shows, or undefined when
 * no session range shows it.
 */
export const findKey = (ref: ranger.Ref, states: State[]): string | undefined =>
  states.find((s) => refsEqual(toRef(s), ref))?.key;

/**
 * @returns the window each reference shows, in order. A key reference resolves to the
 * time range the Core holds for it, and is left out until the Core answers.
 */
export const useResolveWindows = (refs: ranger.Ref[]): Window[] => {
  const keys = refs.filter((r) => r.variant === "key").map((r) => r.key);
  const { data } = Ranger.useResultMultiple(keys.length === 0 ? null : { keys });
  const found = new Map((data ?? []).map((r) => [r.key, r.timeRange]));
  return refs.flatMap((r) => {
    if (r.variant === "window") return [r.window];
    const timeRange = found.get(r.key);
    return timeRange == null ? [] : [staticWindow(timeRange)];
  });
};
