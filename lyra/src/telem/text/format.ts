// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type TimeSpan, type TimeStamp } from "@synnaxlabs/x";

const DAY_MS = 86_400_000;

const localMidnight = (date: Date): number =>
  new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime();

/**
 * Names the local day of `ts` the way a person would relative to `now`: `Today`,
 * `Yesterday`, `Tomorrow`, a weekday within six days, else `Aug 21`, with the year
 * appended only when it differs from the current one.
 */
export const describeDay = (ts: TimeStamp, now: TimeStamp): string => {
  const date = ts.date();
  const days = Math.round((localMidnight(date) - localMidnight(now.date())) / DAY_MS);
  if (days === 0) return "Today";
  if (days === -1) return "Yesterday";
  if (days === 1) return "Tomorrow";
  if (Math.abs(days) <= 6)
    return date.toLocaleDateString(undefined, { weekday: "long" });
  return date.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: ts.localYear === now.localYear ? undefined : "numeric",
  });
};

/**
 * The local time of day: `14:05`, then `:32` when seconds are set, then the fractional
 * groups of `TimeStamp.toPreciseString` when they are set. Digits below `resolution`
 * are dropped first.
 */
export const formatTime = (ts: TimeStamp, resolution?: TimeSpan): string => {
  const shown = resolution == null ? ts : ts.truncate(resolution);
  const full = shown.toPreciseString("local").slice(11);
  return full.endsWith(":00") ? full.slice(0, 5) : full;
};
