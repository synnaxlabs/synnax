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
import {
  type CrudeTimeRange,
  type NumericTimeRange,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { useEffect, useLayoutEffect, useState } from "react";

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

// setTimeout overflows past this delay and fires at once.
const MAX_TIMEOUT_MS = 2 ** 31 - 1;

/**
 * @returns the time to read the stage of `timeRange` at. It holds across renders and
 * advances when now crosses one of the range's ends or the range changes.
 */
export const useClock = (timeRange: CrudeTimeRange): TimeStamp => {
  const { start, end } = new TimeRange(timeRange).makeValid().numeric;
  const [clock, setClock] = useState(() => ({ start, end, now: TimeStamp.now() }));
  useLayoutEffect(
    () =>
      setClock((prev) =>
        prev.start === start && prev.end === end
          ? prev
          : { start, end, now: TimeStamp.now() },
      ),
    [start, end],
  );
  const { now } = clock;
  let boundary: number | null = null;
  if (start < UNSET && start > now.nanoseconds) boundary = start;
  else if (end < UNSET && end > now.nanoseconds) boundary = end;
  useEffect(() => {
    if (boundary == null) return;
    const remaining = (boundary - TimeStamp.now().nanoseconds) / 1e6;
    const delay = Math.min(Math.max(0, Math.ceil(remaining)) + 1, MAX_TIMEOUT_MS);
    const t = setTimeout(
      () => setClock((prev) => ({ ...prev, now: TimeStamp.now() })),
      delay,
    );
    return () => clearTimeout(t);
  }, [boundary, now]);
  return now;
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
