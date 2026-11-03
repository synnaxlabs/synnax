// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/time/Time.css";

import { TimeSpan as XTimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { useNow } from "@/hooks";
import { type BaseProps, Editor, type EditorProps } from "@/input/time/Editor";
import { roundNumeric } from "@/input/time/grammar";
import { suggestTimeSpans } from "@/input/time/suggest";
import { useLanguage } from "@/input/time/useLanguage";
import { type Control } from "@/input/types";
import { Text } from "@/text";

export interface TimeSpanProps
  extends
    Control<number | undefined, number>,
    BaseProps,
    Pick<EditorProps<XTimeSpan>, "inline" | "check" | "onClear"> {
  /**
   * When set, the input shows the time elapsed since this instant, ticking every
   * second, instead of `value`. Typing a duration still commits through `onChange`.
   */
  elapsedSince?: number;
  /** The finest unit the label shows; the tooltip and editor keep every digit. */
  resolution?: XTimeSpan;
  /** The unit of a number typed alone. Defaults to seconds. */
  unit?: XTimeSpan;
  /** What the input says while it has no value. */
  placeholder?: string;
}

/** Formats a span for the label; zero reads as `0s`. */
export const formatTimeSpan = (span: XTimeSpan): string =>
  span.toString("full") || "0s";

const HINT = "Try 2h 30min, 1:30:00, 90, or an hour and a half";
const SHORT_HINT = "Try 500ms, 2.5s, or 1min 30s";

// The clock always reads to the second, whatever the row's labels resolve to: a
// coarser cut would show 0s for the first minute of a long range.
const useElapsed = (since: number | undefined): string | null => {
  const now = useNow((now) => (since == null ? null : now.add(XTimeSpan.SECOND)));
  if (since == null) return null;
  return formatTimeSpan(now.span(since).truncate(XTimeSpan.SECOND));
};

/**
 * A duration input. The value is nanoseconds. At rest it reads `2h 30min 12s`; clicking
 * opens an editor that accepts any duration the grammar does (`30s`, `1:30:00`,
 * `1.5h`) and commits on Enter or a click outside. An inline input is typed in place,
 * with the readings and a hint under it. Unlisted props go to the trigger.
 */
export const TimeSpan = ({
  value,
  onChange,
  elapsedSince,
  resolution,
  unit = XTimeSpan.SECOND,
  placeholder,
  className,
  tooltip,
  ...rest
}: TimeSpanProps): ReactElement => {
  const elapsed = useElapsed(elapsedSince);

  const exact = value == null ? null : new XTimeSpan(roundNumeric(value));
  const formatted = exact == null ? "" : formatTimeSpan(exact);
  const shown =
    exact == null || resolution == null
      ? formatted
      : formatTimeSpan(exact.truncate(resolution));
  const exactTooltip = elapsed == null && shown !== formatted ? formatted : undefined;
  // A unit finer than a second sets short spans, so the examples start short.
  const short = unit.lessThan(XTimeSpan.SECOND);

  const language = useLanguage();
  const suggest = (text: string) => suggestTimeSpans(text, language, unit);

  const handleCommit = (span: XTimeSpan): void => {
    const next = Number(span.valueOf());
    // An elapsed readout ignores value, so any typed duration is a change.
    if (
      elapsedSince != null ||
      value == null ||
      roundNumeric(next) !== roundNumeric(value)
    )
      onChange(next);
  };

  return (
    <Editor<XTimeSpan>
      label={
        <span className={CSS.BE("timespan", "label")}>
          {elapsed ?? (exact == null ? placeholder : shown)}
        </span>
      }
      tooltip={tooltip ?? exactTooltip}
      className={CSS.cls(
        CSS.B("timespan"),
        elapsed != null && CSS.M("elapsed"),
        className,
      )}
      // An elapsed readout is a clock, not a value; the user types a fresh duration.
      initialText={elapsed == null ? formatted : ""}
      suggest={suggest}
      onCommit={handleCommit}
      hint={short ? SHORT_HINT : HINT}
      unreadMessage="Not a duration"
      fieldPlaceholder={elapsed ?? placeholder ?? (short ? "500ms" : "2h 30min")}
      {...rest}
    >
      {({ value: span, reading }) => (
        <>
          <span>{formatTimeSpan(span)}</span>
          <Text.Text
            level="small"
            color={9}
            overflow="fade"
            className={CSS.BE("time-editor", "detail")}
          >
            {reading}
          </Text.Text>
        </>
      )}
    </Editor>
  );
};
