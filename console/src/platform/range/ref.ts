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
import { type NumericTimeRange, type telem, TimeRange } from "@synnaxlabs/x";
import { useEffect, useMemo, useRef } from "react";

import { resolveWindow } from "@/platform/range/resolve";
import { type Session } from "@/session";

const toStringTimeRange = ({ start, end }: NumericTimeRange): telem.StaticWindow => ({
  variant: "static",
  timeRange: {
    start: BigInt(Math.trunc(start)).toString(),
    end: BigInt(Math.trunc(end)).toString(),
  },
});

/**
 * @returns the reference a visualization stores for a session range. A range only the
 * session holds becomes its window, since no other session could resolve its key.
 */
export const toRef = (state: Session.Range.State): ranger.Ref => {
  if (state.variant === "persisted") return { variant: "key", key: state.key };
  if (state.variant === "dynamic")
    return { variant: "window", window: { variant: "dynamic", timeSpan: state.span } };
  return { variant: "window", window: toStringTimeRange(state.timeRange) };
};

/**
 * @returns the fixed window with one bound moved to value. A bound moved across the
 * other carries it along, keeping the window's span.
 */
export const setBound = (
  window: telem.StaticWindow,
  bound: "start" | "end",
  value: number,
): telem.StaticWindow => {
  const start = Number(window.timeRange.start);
  const end = Number(window.timeRange.end);
  const span = end - start;
  if (bound === "start")
    return toStringTimeRange({ start: value, end: value < end ? end : value + span });
  return toStringTimeRange({ start: value > start ? start : value - span, end: value });
};

/**
 * @returns the window a Pluto visualization reads for a reference. While a saved range
 * loads, it holds the last window it returned, so a range switch does not first empty
 * the visualization.
 */
export const useResolveRef = (ref: ranger.Ref): LinePlot.ResolvedRange | undefined => {
  const result = Ranger.useResult(ref.variant === "key" ? { key: ref.key } : null);
  const last = useRef<LinePlot.ResolvedRange | undefined>(undefined);
  const resolved = useMemo((): LinePlot.ResolvedRange | undefined => {
    if (ref.variant === "window") return resolveWindow(ref.window);
    if (result.variant === "success")
      return { variant: "static", timeRange: new TimeRange(result.data.timeRange) };
    return result.variant === "loading" ? last.current : undefined;
  }, [ref, result.variant, result.data]);
  useEffect(() => {
    last.current = resolved;
  }, [resolved]);
  return resolved;
};
