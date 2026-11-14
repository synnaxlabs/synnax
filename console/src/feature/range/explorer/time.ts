// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { Store } from "@synnaxlabs/lyra/store";
import { Flux, Ranger } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";

const hasFields = (where: ranger.Where): boolean =>
  Object.values(where).some((value) => value !== undefined);

/**
 * @returns the filter with its stages and recent spans read at the instant, so the
 * Core answers it as of that instant. A filter without them comes back as it is.
 */
export const pin = (where: ranger.Where, at: TimeStamp): ranger.Where => {
  const { stages, recent, and, or, not, ...rest } = where;
  const base: ranger.Where = { ...rest };
  if (and != null) base.and = and.map((node) => pin(node, at));
  if (or != null) base.or = or.map((node) => pin(node, at));
  if (not != null) base.not = pin(not, at);
  const pinned: ranger.Where[] = [];
  if (stages != null)
    pinned.push({ or: stages.map((stage) => Ranger.stageWhere(stage, at)) });
  if (recent != null)
    pinned.push({ overlaps: new TimeRange(at.sub(new TimeSpan(recent)), at) });
  if (pinned.length === 0) return base;
  return { and: hasFields(base) ? [base, ...pinned] : pinned };
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
  const starting = useFirst(timed ? Ranger.stageWhere("to_do", at) : null, STARTS);
  const ending = useFirst(timed ? Ranger.stageWhere("in_progress", at) : null, ENDS);
  return [...starting, ...ending];
};
