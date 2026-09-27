// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot } from "@synnaxlabs/client";
import { TimeRange, TimeSpan } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Session } from "@/session";

/** The rolling window a new plot shows when the session has no range selected. */
export const DEFAULT_ROLLING = Number(TimeSpan.seconds(30));

/**
 * @returns the plot range that shows the session's range. A static range is copied in,
 * so the plot shows it on machines that lack the session.
 */
export const fromSession = (range: Session.Range.State): lineplot.Range => {
  if (range.variant === "persisted") return { variant: "persisted", key: range.key };
  const { start, end } = new TimeRange(range.timeRange);
  return {
    variant: "static",
    key: range.key,
    name: range.name,
    start: start.valueOf().toString(),
    end: end.valueOf().toString(),
  };
};

/**
 * @returns a function giving the x1 ranges of a new plot: the session's selected range,
 * or {@link DEFAULT_ROLLING} when none is selected.
 */
export const useGetDefaultRanges = (): (() => lineplot.New["ranges"]) => {
  const getState = Session.Range.useGetState();
  return useCallback(() => {
    const selected = getState();
    return {
      x1:
        selected == null
          ? { rolling: DEFAULT_ROLLING, ranges: [] }
          : { ranges: [fromSession(selected)] },
    };
  }, [getState]);
};
