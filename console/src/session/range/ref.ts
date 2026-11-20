// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { type LinePlot, Ranger } from "@synnaxlabs/pluto";
import {
  type color,
  dynamicWindow,
  staticWindow,
  TimeSpan,
  TimeStamp,
  type Window,
} from "@synnaxlabs/x";

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

const windowKey = (w: Window): string =>
  w.variant === "dynamic"
    ? `window:${w.timeSpan}`
    : `window:${w.timeRange.start}:${w.timeRange.end}`;

/** @returns the key that identifies a reference among the ranges of a plot. */
export const keyOf = (ref: ranger.Ref): string =>
  ref.variant === "key" ? ref.key : windowKey(ref.window);

const windowLabel = (w: Window): string =>
  w.variant === "dynamic"
    ? `Rolling ${new TimeSpan(w.timeSpan).toString("semantic")}`
    : TimeStamp.z.parse(w.timeRange.start).toString("preciseDate", "local");

/**
 * @returns the range a line plot shows for each reference, in order. A key reference is
 * left out until the Core answers. A range zeroes at its custom T0 or its start, and a
 * rolling window has no zero. colors holds the color of each reference, in order.
 */
export const useResolveRanges = (
  refs: ranger.Ref[],
  colors: color.Color[],
): LinePlot.RangeSpec[] => {
  const keys = refs.filter((r) => r.variant === "key").map((r) => r.key);
  const { data } = Ranger.useResultMultiple(keys.length === 0 ? null : { keys });
  const found = new Map((data ?? []).map((r) => [r.key, r]));
  return refs.flatMap((r, i): LinePlot.RangeSpec[] => {
    if (r.variant === "window") {
      const { window } = r;
      return [
        {
          key: windowKey(window),
          window,
          label: windowLabel(window),
          color: colors[i],
          zero:
            window.variant === "static"
              ? TimeStamp.z.parse(window.timeRange.start)
              : undefined,
        },
      ];
    }
    const range = found.get(r.key);
    if (range == null) return [];
    const t0 = r.t0 == null ? undefined : new TimeStamp(r.t0);
    return [
      {
        key: r.key,
        window: staticWindow(range.timeRange),
        label: range.name,
        color: colors[i],
        zero: t0 ?? range.timeRange.start,
        customZero: t0 != null,
      },
    ];
  });
};
