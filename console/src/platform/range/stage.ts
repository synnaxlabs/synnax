// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Ranger } from "@synnaxlabs/pluto";
import { type TimeRange, TimeStamp } from "@synnaxlabs/x";

/** Whether the time range has a start, and so a time to show. */
export const scheduled = (timeRange: TimeRange): boolean =>
  !timeRange.makeValid().start.equals(TimeStamp.MAX);

/** Starts a planned range now. It keeps an end that is still ahead. */
export const useStart = (): ((range: ranger.Range) => void) => {
  const { update } = Ranger.useSetTimeRange();
  return (range) =>
    update({
      key: range.key,
      timeRange: Ranger.moveToStage(range.timeRange.numeric, "in_progress"),
    });
};

/** Stops a running range now. */
export const useStop = (): ((range: ranger.Range) => void) => {
  const { update } = Ranger.useSetTimeRange();
  return (range) =>
    update({
      key: range.key,
      timeRange: Ranger.moveToStage(range.timeRange.numeric, "completed"),
    });
};

/** The icon for Next: stop a running range and start the one after it. */
export const NextIcon = Icon.createFromPath(
  "next",
  "M6 18l8.5-6L6 6v12zM16 6v12h2V6h-2z",
);
