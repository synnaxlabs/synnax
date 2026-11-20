// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ranger } from "@synnaxlabs/client";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Store } from "@synnaxlabs/lyra/store";
import {
  type CrudeTimeRange,
  type NumericTimeRange,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { UNSET } from "@/ranger/move";

/** @returns the filter that matches the ranges `getStage` puts in `stage` at `now`. */
export const stageWhere = (stage: ranger.Stage, now: TimeStamp): ranger.Where => {
  const next = now.add(TimeSpan.nanoseconds(1));
  if (stage === "to_do")
    return { not: { overlaps: new TimeRange(TimeStamp.MIN, next) } };
  if (stage === "in_progress") return { overlaps: new TimeRange(now, next) };
  return { not: { overlaps: new TimeRange(now, TimeStamp.MAX) } };
};

/** @returns the stage `timeRange` is in at `now`. */
export const getStage = (
  timeRange: CrudeTimeRange,
  now: TimeStamp = TimeStamp.now(),
): ranger.Stage => {
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

/** @returns the recent spans of the filter, or null when it reads no time. */
const timeSpans = (where: ranger.Where): TimeSpan[] | null => {
  const children = [...(where.and ?? []), ...(where.or ?? [])];
  if (where.not != null) children.push(where.not);
  let reads = where.stages != null;
  const spans = where.recent == null ? [] : [new TimeSpan(where.recent)];
  for (const child of children) {
    const nested = timeSpans(child);
    if (nested == null) continue;
    reads = true;
    spans.push(...nested);
  }
  return reads || spans.length > 0 ? spans : null;
};

/**
 * @returns the first instant after `at` when one of the ranges starts, ends, or moves
 * past a recent span of the filter, or null when the filter reads no time.
 */
export const nextCrossing = (
  where: ranger.Where | undefined,
  ranges: ranger.Range[],
  at: TimeStamp,
): TimeStamp | null => {
  const spans = where == null ? null : timeSpans(where);
  if (spans == null) return null;
  let next: TimeStamp | null = null;
  for (const { timeRange } of ranges)
    for (const bound of [timeRange.start, timeRange.end]) {
      if (bound.equals(TimeStamp.MAX)) continue;
      for (const instant of [bound, ...spans.map((span) => bound.add(span))])
        if (instant.after(at) && (next == null || instant.before(next))) next = instant;
    }
  return next;
};

const STARTS: ranger.Order = { field: "time_range_start", descending: false };
const ENDS: ranger.Order = { field: "time_range_end", descending: false };

const useFirst = (where: ranger.Where | null, order: ranger.Order): ranger.Range[] => {
  const list = Flux.useList((client) => client.ranges, {
    name: "ranges",
    query: where == null ? null : { where, order },
    pageSize: 1,
    eager: true,
  });
  return Store.useItems({
    keys: list.data,
    subscribe: list.subscribe,
    getItem: list.getItem,
  });
};

/**
 * @returns the next range to start and the next running range to end, read at `at`,
 * or none when the filter reads no time. A created or changed range updates them, so
 * their crossings catch a range that enters the filter after the list reads.
 */
export const useNextRanges = (
  where: ranger.Where | undefined,
  at: TimeStamp,
): ranger.Range[] => {
  const timed = where != null && timeSpans(where) != null;
  const starting = useFirst(timed ? stageWhere("to_do", at) : null, STARTS);
  const ending = useFirst(timed ? stageWhere("in_progress", at) : null, ENDS);
  return [...starting, ...ending];
};

export const STAGE_ICONS: Record<ranger.Stage, Icon.FC> = {
  to_do: Icon.ToDo,
  in_progress: Icon.InProgress,
  completed: Icon.Completed,
};

export const STAGE_NAMES: Record<ranger.Stage, string> = {
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
  stage: ranger.Stage,
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
