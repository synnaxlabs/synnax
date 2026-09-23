// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/time/Time.css";

import {
  type NumericTimeRange,
  type TimeSpan as XTimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useCallback } from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { type Action, type BaseProps, Editor } from "@/input/time/Editor";
import {
  type Anchors,
  fromNumeric,
  nudge,
  parseTimeStamp,
  roundNumeric,
  unitAt,
  zoneName,
} from "@/input/time/grammar";
import { type Bound, suggestTimeStamps } from "@/input/time/suggest";
import { useLanguage } from "@/input/time/useLanguage";
import { type Control } from "@/input/types";
import { Text as TelemText } from "@/telem/text";
import { Text } from "@/text";

/** The instants an input may anchor typed expressions on, as form values. */
export interface DateTimeAnchors {
  /** The other end of the range the input belongs to. */
  start?: number;
  end?: number;
  /** The parent range, the `T` in `T+3.2s`. */
  parent?: NumericTimeRange;
}

export interface DateTimeProps extends Control<number>, BaseProps {
  anchors?: DateTimeAnchors;
  /**
   * An instant whose day is already shown beside this input; the label drops its own
   * day when the two match.
   */
  sharedDay?: number;
  /** The end of a range the input holds; decides what a bare duration or time means. */
  bound?: Bound;
  /**
   * Rendered under the options with the value the highlighted reading or the hovered
   * action would commit. Use it to say what else a commit would change, and return
   * null when nothing else would.
   */
  effect?: Component.RenderProp<{ candidate: number }>;
  /** The finest unit the label shows; the tooltip and editor keep every digit. */
  resolution?: XTimeSpan;
  /**
   * Renders a value equal to this as an empty input and commits it when the field is
   * cleared.
   */
  emptyValue?: number;
  /** What the empty input says at rest, as a prompt: `Set a start time`. */
  placeholder?: string;
  /** The action that clears the input back to `emptyValue`. */
  clearLabel?: string;
}

const resolveAnchors = (
  start: number | undefined,
  end: number | undefined,
  parentStart: number | undefined,
): Anchors => {
  const anchors: Anchors = { now: TimeStamp.now() };
  // An end at the edge of time is open, not an instant to offset from.
  if (start != null && start > TimeStamp.MIN.nanoseconds)
    anchors.start = fromNumeric(start);
  if (end != null && end < TimeStamp.MAX.nanoseconds) anchors.end = fromNumeric(end);
  if (parentStart != null) anchors.parent = fromNumeric(parentStart);
  return anchors;
};

const describeOffset = (value: TimeStamp, from: TimeStamp): string => {
  const span = value.span(from);
  if (span.isZero) return "";
  const text = span.toString("semantic");
  return value.after(from) ? `${text} from now` : `${text} ago`;
};

const describeParent = (value: TimeStamp, parent: TimeStamp): string => {
  const sign = value.before(parent) ? "-" : "+";
  return `T${sign}${value.span(parent).toString("full") || "0s"}`;
};

const FIXED_LAYOUT_RE =
  /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d{3}(?: \d{3}){0,2})?$/;

/**
 * A date-time input. The value is nanoseconds since the Unix epoch. At rest it reads
 * as content (`Today 14:05:32`); clicking opens an editor with the instant in a fixed
 * local layout, where typing replaces it with an expression and Up and Down nudge
 * the unit under the caret (Shift for ten). Enter or a click outside commits. Unlisted
 * props go to the trigger.
 */
export const DateTime = ({
  value,
  onChange,
  anchors: { start, end, parent } = {},
  emptyValue,
  placeholder,
  sharedDay,
  bound,
  effect,
  resolution,
  clearLabel = "Clear",
  className,
  tooltip,
  ...rest
}: DateTimeProps): ReactElement => {
  const isEmpty = emptyValue != null && value === emptyValue;
  const stamp = isEmpty ? null : fromNumeric(value);
  const formatted = stamp?.toPreciseString("local") ?? "";
  const parentStart = parent?.start;
  const languageReady = useLanguage();

  // Editor memoizes the readings on this function. Anchors resolve on each call, so
  // `now` inside a reading is the moment of the edit.
  const suggest = useCallback(
    (text: string) =>
      suggestTimeStamps(text, {
        anchors: resolveAnchors(start, end, parentStart),
        current: isEmpty ? undefined : fromNumeric(value),
        bound,
      }),
    [start, end, parentStart, isEmpty, value, bound, languageReady],
  );

  // Digits in the fixed layout nudge; a phrase walks its readings.
  const nudgeText = (text: string, caret: number, steps: number): string | null => {
    if (!FIXED_LAYOUT_RE.test(text.trim())) return null;
    const parsed = parseTimeStamp(text, resolveAnchors(start, end, parentStart));
    if (!parsed.ok) return null;
    return nudge(parsed.value, unitAt(caret), steps).toPreciseString("local");
  };

  const commit = (next: number): void => {
    // The empty value is a sentinel, not an instant to round.
    const round = (v: number): number =>
      v === emptyValue ? v : Number(roundNumeric(v));
    if (round(next) !== round(value)) onChange(round(next));
  };

  const actions: Action<TimeStamp>[] = [
    { key: "now", icon: <Icon.Time />, label: "Now", value: () => TimeStamp.now() },
  ];
  if (emptyValue != null && !isEmpty)
    actions.push({
      key: "empty",
      icon: <Icon.Close />,
      label: clearLabel,
      value: () => new TimeStamp(emptyValue),
    });
  if (parent != null) {
    actions.push({
      key: "parentStart",
      icon: <Icon.Range />,
      label: "Parent start",
      hint: "T+0",
      value: () => fromNumeric(parent.start),
    });
    if (parent.end < TimeStamp.MAX.nanoseconds)
      actions.push({
        key: "parentEnd",
        icon: <Icon.Range />,
        label: "Parent end",
        value: () => fromNumeric(parent.end),
      });
  }

  let label: ReactNode;
  if (stamp == null)
    label = <span className={CSS.BE("datetime", "empty")}>{placeholder}</span>;
  else {
    const showDay = sharedDay == null || !stamp.isSameDay(sharedDay, "local");
    label = (
      <>
        {showDay && (
          <span className={CSS.BE("datetime", "day")}>
            {TelemText.describeDay(stamp, TimeStamp.now())}
          </span>
        )}
        <span>{TelemText.formatTime(stamp, resolution)}</span>
      </>
    );
  }

  const exactTooltip = stamp == null ? undefined : `${formatted} ${zoneName(stamp)}`;

  return (
    <Editor<TimeStamp>
      label={label}
      tooltip={tooltip ?? exactTooltip}
      className={CSS.cls(CSS.B("datetime"), className)}
      initialText={formatted}
      suggest={suggest}
      nudge={nudgeText}
      onCommit={(next) => commit(Number(next.valueOf()))}
      onClear={emptyValue == null ? undefined : () => commit(emptyValue)}
      hint="Try 14:05, tomorrow 3pm, now - 5m, or 2h"
      unreadMessage="Not a time"
      fieldPlaceholder="14:05, tomorrow 3pm, now - 5m"
      actions={actions}
      effect={
        effect == null
          ? undefined
          : ({ candidate }) => effect({ candidate: Number(candidate.valueOf()) })
      }
      {...rest}
    >
      {({ value: candidate, reading }) => {
        const now = TimeStamp.now();
        // A reading relative to now already says what the offset would.
        const details = [reading];
        if (!/\b(now|ago)$/.test(reading)) details.push(describeOffset(candidate, now));
        if (parentStart != null)
          details.push(describeParent(candidate, fromNumeric(parentStart)));
        return (
          <>
            <span>
              <span className={CSS.BE("datetime", "day")}>
                {TelemText.describeDay(candidate, now)}
              </span>{" "}
              {TelemText.formatTime(candidate)}
            </span>
            <Text.Text
              level="small"
              color={9}
              overflow="ellipsis"
              className={CSS.BE("time-editor", "detail")}
            >
              {details.filter((d) => d.length > 0).join(" · ")}
            </Text.Text>
          </>
        );
      }}
    </Editor>
  );
};
