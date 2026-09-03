// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Input, LinePlot } from "@synnaxlabs/pluto";
import { TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

const SPAN_UNITS: Record<string, (value: number) => TimeSpan> = {
  ns: (value) => TimeSpan.nanoseconds(value),
  us: (value) => TimeSpan.microseconds(value),
  µs: (value) => TimeSpan.microseconds(value),
  ms: (value) => TimeSpan.milliseconds(value),
  s: (value) => TimeSpan.seconds(value),
  m: (value) => TimeSpan.minutes(value),
  h: (value) => TimeSpan.hours(value),
  d: (value) => TimeSpan.days(value),
  w: (value) => TimeSpan.days(7 * value),
  mo: (value) => TimeSpan.days(30 * value),
  y: (value) => TimeSpan.days(365 * value),
};

// Two-letter units first so "ms" never parses as minutes then seconds.
const UNIT = "mo|ms|ns|us|µs|[smhdwy]";
const SPAN_TOKEN = new RegExp(`(\\d+(?:\\.\\d+)?)\\s*(${UNIT})`, "g");
const SPAN_VALID = new RegExp(`^(?:\\s*\\d+(?:\\.\\d+)?\\s*(?:${UNIT}))+\\s*$`);

/** Parses a duration like "45m" or "1h 30m" into nanoseconds. */
export const parseSpan = (input: string): number | null => {
  const text = input.toLowerCase();
  if (!SPAN_VALID.test(text)) return null;
  let total = 0;
  for (const [, value, unit] of text.matchAll(SPAN_TOKEN))
    total += Number(SPAN_UNITS[unit](parseFloat(value)));
  return total > 0 ? total : null;
};

/** Edits the rolling window the plot's "custom" range key resolves to. */
export const CustomRangeInput = (): ReactElement => {
  const custom = LinePlot.useCustomRange();
  const dispatch = LinePlot.useSingleDispatch();
  const handleChange = useCallback(
    (raw: string) => {
      const span = parseSpan(raw);
      if (span == null) return;
      dispatch(lineplot.setCustomRange({ custom: { variant: "dynamic", span } }));
    },
    [dispatch],
  );
  const value =
    custom?.variant === "dynamic" ? new TimeSpan(custom.span).toString() : "";
  return (
    <Input.Item x label="Custom">
      <Input.Text
        value={value}
        onChange={handleChange}
        onlyChangeOnBlur
        resetOnBlurIfEmpty
        placeholder="1h 30m"
        grow
      />
    </Input.Item>
  );
};
