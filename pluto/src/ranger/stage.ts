// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import {
  type CrudeTimeRange,
  type NumericTimeRange,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { UNSET } from "@/ranger/move";

export const STAGES = ["to_do", "in_progress", "completed"] as const;

export type Stage = (typeof STAGES)[number];

export const sortByStage = (a: ranger.Range, b: ranger.Range): number =>
  STAGES.indexOf(getStage(a.timeRange)) - STAGES.indexOf(getStage(b.timeRange));

/** @returns the stage `timeRange` is in at `now`. */
export const getStage = (
  timeRange: CrudeTimeRange,
  now: TimeStamp = TimeStamp.now(),
): Stage => {
  const tr = new TimeRange(timeRange).makeValid();
  if (now.before(tr.start)) return "to_do";
  if (now.afterEq(tr.end)) return "completed";
  return "in_progress";
};

/**
 * @returns the next instant after `now` that changes how the range reads: one of its
 * ends, a day past the start of an open range, or local midnight.
 */
const nextBoundary = (start: number, end: number, now: TimeStamp): TimeStamp => {
  const midnight = now.date();
  midnight.setHours(24, 0, 0, 0);
  let next = new TimeStamp(midnight);
  const day = end >= UNSET ? start + Number(TimeSpan.DAY.valueOf()) : UNSET;
  for (const instant of [start, end, day]) {
    if (instant >= UNSET) continue;
    // Compares as getStage does. A float64 compare reads an instant that rounds up past
    // the clock's reading as passed, so the stage never moves.
    const ts = new TimeStamp(instant);
    if (ts.after(now) && ts.before(next)) next = ts;
  }
  return next;
};

/**
 * @returns the time to read the stage and labels of `timeRange` at. It holds across
 * renders and advances when it crosses an instant that changes how the range reads:
 * one of its ends, a day past the start of an open range, or local midnight.
 */
export const useClock = (timeRange: CrudeTimeRange): TimeStamp => {
  const { start, end } = new TimeRange(timeRange).makeValid().numeric;
  return useNow((now) => nextBoundary(start, end, now));
};

export const STAGE_ICONS: Record<Stage, Icon.FC> = {
  to_do: Icon.ToDo,
  in_progress: Icon.InProgress,
  completed: Icon.Completed,
};

export const STAGE_NAMES: Record<Stage, string> = {
  to_do: "To do",
  in_progress: "In progress",
  completed: "Completed",
};

/**
 * Returns the range with the timestamps that put it in `stage` as of `at`: to do moves
 * a past start to the end, in progress stamps a future start and clears a past end,
 * and completed stamps a future end.
 */
export const moveToStage = (
  value: NumericTimeRange,
  stage: Stage,
  at: TimeStamp = TimeStamp.now(),
): NumericTimeRange => {
  // We subtract a millisecond here to avoid weird issues where you select "completed"
  // but you actually get "in_progress" or "to_do" because of precision issues with
  // numeric time ranges.
  const now = at.sub(TimeSpan.MILLISECOND).nanoseconds;
  const tr = new TimeRange(value).makeValid().numeric;
  switch (stage) {
    case "to_do":
      if (tr.end <= now) tr.end = TimeStamp.MAX.nanoseconds;
      if (tr.start <= now) tr.start = tr.end;
      break;
    case "in_progress":
      if (tr.start > now) tr.start = now;
      if (tr.end <= now) tr.end = TimeStamp.MAX.nanoseconds;
      break;
    case "completed":
      if (tr.end > now) tr.end = now;
      if (tr.start > tr.end) tr.start = tr.end;
      break;
  }
  return tr;
};
