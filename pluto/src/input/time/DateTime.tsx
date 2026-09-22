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
import { type ReactElement, type ReactNode, useCallback, useMemo } from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { Flex } from "@/flex";
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
   * Rendered once in every reading and action with the value it would commit. Use it
   * to say what else a commit would change.
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

const resolveAnchors = ({ start, end, parent }: DateTimeAnchors): Anchors => {
  const anchors: Anchors = { now: TimeStamp.now() };
  // An end at the edge of time is open, not an instant to offset from.
  if (start != null && start > TimeStamp.MIN.nanoseconds)
    anchors.start = new TimeStamp(start);
  if (end != null && end < TimeStamp.MAX.nanoseconds) anchors.end = new TimeStamp(end);
  if (parent != null) anchors.parent = new TimeStamp(parent.start);
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
  const anchors = resolveAnchors({ start, end, parent });

  // Anchors resolve on each call, so `now` inside a reading is the moment of the edit.
  const suggest = useCallback(
    (text: string) =>
      suggestTimeStamps(text, {
        anchors: resolveAnchors({ start, end, parent }),
        current: isEmpty ? undefined : fromNumeric(value),
        bound,
      }),
    [start, end, parent, isEmpty, value, bound],
  );

  // Digits in the fixed layout nudge; a phrase walks its readings.
  const nudgeText = useCallback(
    (text: string, caret: number, steps: number): string | null => {
      if (!FIXED_LAYOUT_RE.test(text.trim())) return null;
      const parsed = parseTimeStamp(text, resolveAnchors({ start, end, parent }));
      if (!parsed.ok) return null;
      return nudge(parsed.value, unitAt(caret), steps).toPreciseString("local");
    },
    [start, end, parent],
  );

  const commit = useCallback(
    (next: number) => {
      // The empty value is a sentinel, not an instant to round.
      const round = (v: number): number =>
        v === emptyValue ? v : Number(roundNumeric(v));
      if (round(next) !== round(value)) onChange(round(next));
    },
    [onChange, value, emptyValue],
  );

  const handleCommit = useCallback(
    (next: TimeStamp) => commit(Number(next.valueOf())),
    [commit],
  );

  const handleClear = useCallback(() => {
    if (emptyValue != null) commit(emptyValue);
  }, [emptyValue, commit]);

  const actions = useMemo(() => {
    const out: Action<TimeStamp>[] = [
      {
        key: "now",
        icon: <Icon.Time />,
        label: "Now",
        hint: "now",
        value: () => TimeStamp.now(),
      },
    ];
    if (emptyValue != null && !isEmpty)
      out.push({
        key: "empty",
        icon: <Icon.Close />,
        label: clearLabel,
        hint: "empty",
        value: () => new TimeStamp(emptyValue),
      });
    if (parent == null) return out;
    out.push({
      key: "parentStart",
      icon: <Icon.Range />,
      label: "Parent start",
      hint: "T+0",
      value: () => new TimeStamp(parent.start),
    });
    if (parent.end < TimeStamp.MAX.nanoseconds)
      out.push({
        key: "parentEnd",
        icon: <Icon.Range />,
        label: "Parent end",
        hint: "T+end",
        value: () => new TimeStamp(parent.end),
      });
    return out;
  }, [emptyValue, isEmpty, clearLabel, parent]);

  const editorEffect = useMemo(
    () =>
      effect == null
        ? undefined
        : ({ candidate }: { candidate: TimeStamp }) =>
            effect({ candidate: Number(candidate.valueOf()) }),
    [effect],
  );

  let label: ReactNode;
  if (stamp == null)
    label = <span className={CSS.BE("datetime", "empty")}>{placeholder}</span>;
  else {
    const showDay = sharedDay == null || !stamp.isSameDay(sharedDay, "local");
    label = (
      <>
        {showDay && (
          <span className={CSS.BE("datetime", "day")}>
            {TelemText.describeDay(stamp, anchors.now)}
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
      onCommit={handleCommit}
      onClear={handleClear}
      hint="Try 14:05, tomorrow 3pm, now - 5m, or 2h"
      unreadMessage="Not a time"
      fieldPlaceholder="14:05, tomorrow 3pm, now - 5m"
      actions={actions}
      effect={editorEffect}
      {...rest}
    >
      {({ value: candidate, reading }) => {
        // A reading relative to now already says what the offset hint would.
        const relative = /\b(now|ago)$/.test(reading);
        const hints = relative ? [] : [describeOffset(candidate, anchors.now)];
        if (anchors.parent != null)
          hints.push(describeParent(candidate, anchors.parent));
        return (
          <>
            <Flex.Box x align="center" justify="between" gap="medium">
              <span>
                <span className={CSS.BE("datetime", "day")}>
                  {TelemText.describeDay(candidate, anchors.now)}
                </span>{" "}
                {TelemText.formatTime(candidate)}
              </span>
              <Text.Text level="small" color={9}>
                {hints.filter((h) => h.length > 0).join(" · ")}
              </Text.Text>
            </Flex.Box>
            <Text.Text level="small" color={9} overflow="ellipsis">
              {reading}
            </Text.Text>
          </>
        );
      }}
    </Editor>
  );
};
