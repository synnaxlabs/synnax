// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import {
  type CrudeTimeRange,
  type NumericTimeRange,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { Icon } from "@/icon";

export const STAGES = ["to_do", "in_progress", "completed"] as const;

export type Stage = (typeof STAGES)[number];

export const sortByStage = (a: ranger.Range, b: ranger.Range): number =>
  STAGES.indexOf(getStage(a.timeRange)) - STAGES.indexOf(getStage(b.timeRange));

export const getStage = (timeRange: CrudeTimeRange): Stage => {
  const tr = new TimeRange(timeRange).makeValid();
  const now = TimeStamp.now();
  if (now.before(tr.start)) return "to_do";
  if (now.after(tr.end)) return "completed";
  return "in_progress";
};

export const STAGE_ICONS: Record<Stage, Icon.FC> = {
  to_do: Icon.ToDo,
  in_progress: Icon.InProgress,
  completed: Icon.Completed,
};

export const STAGE_NAMES: Record<Stage, string> = {
  to_do: "To Do",
  in_progress: "In Progress",
  completed: "Completed",
};

interface WrapNumericTimeRangeToStageArgs {
  value: NumericTimeRange;
  onChange: (value: NumericTimeRange) => void;
}

interface WrapNumericTimeRangeToStageReturn {
  value: Stage;
  onChange: (value: Stage) => void;
}

export const wrapNumericTimeRangeToStage = ({
  value,
  onChange,
}: WrapNumericTimeRangeToStageArgs): WrapNumericTimeRangeToStageReturn => ({
  value: getStage(value),
  onChange: (v: Stage) => {
    // We subtract a millisecond here to avoid weird issues where you select "completed"
    // but you actually get "in_progress" or "to_do" because of precision issues with
    // numeric time ranges.
    const now = TimeStamp.now().sub(TimeSpan.MILLISECOND).nanoseconds;
    const tr = new TimeRange(value).makeValid().numeric;
    switch (v) {
      case "to_do":
        if (tr.end < now) tr.end = TimeStamp.MAX.nanoseconds;
        if (tr.start < now) tr.start = tr.end;
        break;
      case "in_progress":
        if (tr.start > now) tr.start = now;
        if (tr.end < now) tr.end = TimeStamp.MAX.nanoseconds;
        break;
      case "completed":
        if (tr.end > now) tr.end = now;
        if (tr.start > tr.end) tr.start = tr.end;
        break;
    }
    onChange(tr);
  },
});
