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
import { useMemo } from "react";

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

/** @returns the window a Pluto visualization reads for a reference. */
export const useResolveRef = (ref: ranger.Ref): LinePlot.ResolvedRange | undefined => {
  const { data } = Ranger.useResult(ref.variant === "key" ? { key: ref.key } : null);
  return useMemo(() => {
    if (ref.variant === "window") return resolveWindow(ref.window);
    if (data == null) return undefined;
    return { variant: "static", timeRange: new TimeRange(data.timeRange) };
  }, [ref, data]);
};
