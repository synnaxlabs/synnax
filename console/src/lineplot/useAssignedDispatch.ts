// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { LinePlot } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Range } from "@/range";

const ROLLING_30S_RANGE_KEY = "recent";

const CHANNEL_OR_X_TYPES: ReadonlySet<lineplot.Action["type"]> = new Set([
  "add_channel",
  "set_x_channel",
]);

// useAssignedDispatch wraps Pluto's lineplot dispatch with the
// console-only augmentation that the legacy assignActiveRangeEffect
// middleware used to perform: when channels are first bound to a plot
// that has no x1 ranges yet, prepend an AddRange(activeRange) so the
// canvas has something to plot against. The active range is a console
// concept and lives in Redux. Range keys may be synthetic console-side
// strings (e.g. "recent") rather than UUIDs; the lineplot schema accepts
// opaque strings so synthetic ranges round-trip cleanly.
export const useAssignedDispatch = (key: lineplot.Key) => {
  const { dispatch } = LinePlot.useDispatch();
  const ranges = LinePlot.useSelectRanges({ key });
  const activeRange = Range.useSelectActiveKey() ?? ROLLING_30S_RANGE_KEY;
  return useCallback(
    (actions: lineplot.Action[]): void => {
      const needsActiveRange =
        ranges.x1.length === 0 && actions.some((a) => CHANNEL_OR_X_TYPES.has(a.type));
      const finalActions = needsActiveRange
        ? [lineplot.addRange({ axisKey: "x1", range: activeRange }), ...actions]
        : actions;
      if (finalActions.length > 0) dispatch({ key, actions: finalActions });
    },
    [dispatch, key, ranges.x1.length, activeRange],
  );
};
