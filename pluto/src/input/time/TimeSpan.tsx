// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/time/Time.css";

import { TimeSpan as XTimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useState } from "react";

import { CSS } from "@/css";
import { type BaseProps, Editor } from "@/input/time/Editor";
import { roundNumeric } from "@/input/time/grammar";
import { suggestTimeSpans } from "@/input/time/suggest";
import { type Control } from "@/input/types";
import { Text } from "@/text";

export interface TimeSpanProps extends Control<number>, BaseProps {
  /**
   * When set, the input shows the time elapsed since this instant, ticking every
   * second, instead of `value`. Typing a duration still commits through `onChange`.
   */
  elapsedSince?: number;
  /** The finest unit the label shows; the tooltip and editor keep every digit. */
  resolution?: XTimeSpan;
}

/** Formats a span for the label; zero reads as `0s`. */
export const formatTimeSpan = (span: XTimeSpan): string =>
  span.toString("full") || "0s";

const useElapsed = (
  since: number | undefined,
  resolution: XTimeSpan = XTimeSpan.SECOND,
): string | null => {
  const [, setTick] = useState(0);
  useEffect(() => {
    if (since == null) return;
    const i = setInterval(() => setTick((p) => p + 1), 1000);
    return () => clearInterval(i);
  }, [since]);
  if (since == null) return null;
  // The clock ticks by the second, so it never reads finer than that.
  const shown = resolution.lessThan(XTimeSpan.SECOND) ? XTimeSpan.SECOND : resolution;
  return formatTimeSpan(TimeStamp.now().span(since).truncate(shown));
};

/**
 * A duration input. The value is nanoseconds. At rest it reads `2h 30m 12s`; clicking
 * opens an editor that accepts any duration the grammar does (`30s`, `1:30:00`,
 * `1.5h`) and commits on Enter or a click outside. Unlisted props go to the trigger.
 */
export const TimeSpan = ({
  value,
  onChange,
  elapsedSince,
  resolution,
  className,
  tooltip,
  ...rest
}: TimeSpanProps): ReactElement => {
  const elapsed = useElapsed(elapsedSince, resolution);

  const exact = new XTimeSpan(roundNumeric(value));
  const formatted = formatTimeSpan(exact);
  const shown =
    resolution == null ? formatted : formatTimeSpan(exact.truncate(resolution));
  const exactTooltip = elapsed == null && shown !== formatted ? formatted : undefined;

  const handleCommit = useCallback(
    (span: XTimeSpan) => {
      const next = Number(span.valueOf());
      // An elapsed readout ignores value, so any typed duration is a change.
      if (elapsedSince != null || roundNumeric(next) !== roundNumeric(value))
        onChange(next);
    },
    [value, onChange, elapsedSince],
  );

  return (
    <Editor<XTimeSpan>
      label={<span className={CSS.BE("timespan", "label")}>{elapsed ?? shown}</span>}
      tooltip={tooltip ?? exactTooltip}
      className={CSS.cls(
        CSS.B("timespan"),
        elapsed != null && CSS.M("elapsed"),
        className,
      )}
      // An elapsed readout is a clock, not a value; the user types a fresh duration.
      initialText={elapsed == null ? formatted : ""}
      suggest={suggestTimeSpans}
      onCommit={handleCommit}
      hint="Try 2h 30m, 1:30:00, 90, or an hour and a half"
      unreadMessage="Not a duration"
      fieldPlaceholder={elapsed ?? "2h 30m"}
      {...rest}
    >
      {({ value: span, reading }) => (
        <>
          <span>{formatTimeSpan(span)}</span>
          <Text.Text level="small" color={9} overflow="ellipsis">
            {reading}
          </Text.Text>
        </>
      )}
    </Editor>
  );
};
