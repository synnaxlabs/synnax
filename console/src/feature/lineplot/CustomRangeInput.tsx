// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, lineplot } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Channel, LinePlot } from "@synnaxlabs/pluto";
import { TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { FLAGS } from "@/flags";
import { Range } from "@/platform/range";

// SY-4816: Static range entry, when added, should stop at 1s, which also simplifies
// its selector and UX.
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
const NUM = "(?:\\d+(?:\\.\\d+)?|\\.\\d+)";
const SPAN_TOKEN = new RegExp(`(${NUM})(${UNIT})`, "g");
const SPAN_VALID = new RegExp(`^\\s*(?:${NUM}(?:${UNIT})\\s*)+$`);

/** Parses a duration like "45m" or "1h 30m" into nanoseconds. */
export const parseSpan = (input: string): number | null => {
  const text = input.toLowerCase();
  if (!SPAN_VALID.test(text)) return null;
  let total = 0;
  for (const [, value, unit] of text.matchAll(SPAN_TOKEN))
    total += Number(SPAN_UNITS[unit](parseFloat(value)));
  return total > 0 ? total : null;
};

/** The triggered window a plot starts with: one 50 ms frame per rising crossing. */
export const DEFAULT_TRIGGERED: lineplot.TriggeredCustomRange = {
  variant: "triggered",
  channel: 0,
  level: 0,
  edge: "rising",
  span: Number(TimeSpan.milliseconds(50).valueOf()),
  pretrigger: 0.1,
  timeout: Number(TimeSpan.seconds(1).valueOf()),
};

const DEFAULT_DYNAMIC: lineplot.DynamicCustomRange = {
  variant: "dynamic",
  span: Number(TimeSpan.minutes(1).valueOf()),
};

type Variant = "dynamic" | "triggered";

const SelectVariant = (props: Select.ButtonsProps<Variant>): ReactElement => (
  <Select.Buttons<Variant> {...props}>
    <Select.Item itemKey="dynamic" size="small">
      Rolling
    </Select.Item>
    <Select.Item itemKey="triggered" size="small">
      Triggered
    </Select.Item>
  </Select.Buttons>
);

const SelectEdge = (props: Select.ButtonsProps<lineplot.TriggerEdge>): ReactElement => (
  <Select.Buttons<lineplot.TriggerEdge> {...props}>
    <Select.Item itemKey="rising" size="small">
      Rising
    </Select.Item>
    <Select.Item itemKey="falling" size="small">
      Falling
    </Select.Item>
  </Select.Buttons>
);

const PRETRIGGER_BOUNDS = { lower: 0, upper: 1 };
const PRETRIGGER_DRAG_SCALE = { x: 0.001, y: 0.001 };
const LEVEL_DRAG_SCALE = { x: 0.01, y: 0.01 };

interface SpanInputProps extends Omit<Input.TextProps, "value" | "onChange"> {
  value: number;
  onChange: (span: number) => void;
}

const SpanInput = ({ value, onChange, ...rest }: SpanInputProps): ReactElement => {
  const handleChange = useCallback(
    (raw: string) => {
      const span = parseSpan(raw);
      if (span != null) onChange(span);
    },
    [onChange],
  );
  return (
    <Input.Text
      value={new TimeSpan(value).toString()}
      onChange={handleChange}
      onlyChangeOnBlur
      resetOnBlurIfEmpty
      {...rest}
    />
  );
};

interface TriggeredFormProps {
  value: lineplot.TriggeredCustomRange;
  onChange: (value: lineplot.TriggeredCustomRange) => void;
}

const TriggeredForm = ({ value, onChange }: TriggeredFormProps): ReactElement => {
  const set = <K extends keyof lineplot.TriggeredCustomRange>(
    key: K,
    v: lineplot.TriggeredCustomRange[K],
  ) => onChange({ ...value, [key]: v });
  return (
    <>
      <Input.Item x label="Trigger">
        <Channel.SelectSingle
          value={value.channel === 0 ? undefined : value.channel}
          onChange={(ch: channel.Key | null) => set("channel", ch ?? 0)}
          allowNone
          location="top"
        />
      </Input.Item>
      <Input.Item x label="Level">
        <Input.Numeric
          value={value.level}
          onChange={(v: number) => set("level", v)}
          dragScale={LEVEL_DRAG_SCALE}
        />
      </Input.Item>
      <Input.Item x label="Edge">
        <SelectEdge
          value={value.edge}
          onChange={(v: lineplot.TriggerEdge) => set("edge", v)}
        />
      </Input.Item>
      <Input.Item x label="Span">
        <SpanInput
          value={value.span}
          onChange={(v) => set("span", v)}
          placeholder="50ms"
        />
      </Input.Item>
      <Input.Item x label="Pre-trigger">
        <Input.Numeric
          value={value.pretrigger}
          onChange={(v: number) => set("pretrigger", v)}
          bounds={PRETRIGGER_BOUNDS}
          dragScale={PRETRIGGER_DRAG_SCALE}
        />
      </Input.Item>
      <Input.Item x label="Timeout">
        <SpanInput
          value={value.timeout}
          onChange={(v) => set("timeout", v)}
          placeholder="1s"
        />
      </Input.Item>
    </>
  );
};

export interface CustomRangeInputProps {
  axisKey: lineplot.XAxisKey;
}

/** Edits the window the plot's "custom" range key resolves to. */
export const CustomRangeInput = ({ axisKey }: CustomRangeInputProps): ReactElement => {
  const custom = LinePlot.useCustomRange();
  const dispatch = LinePlot.useSingleDispatch();
  const handleSpanChange = useCallback(
    (span: number) =>
      dispatch(lineplot.setCustomRange({ custom: { variant: "dynamic", span } })),
    [dispatch],
  );
  const handleVariantChange = useCallback(
    (variant: Variant) => {
      if (variant === "dynamic")
        dispatch(lineplot.setCustomRange({ custom: DEFAULT_DYNAMIC }));
      else {
        // A one-frame window cannot share an axis with a range.
        dispatch(lineplot.setRanges({ axisKey, ranges: [Range.CUSTOM_KEY] }));
        dispatch(lineplot.setCustomRange({ custom: DEFAULT_TRIGGERED }));
      }
    },
    [dispatch, axisKey],
  );
  const handleTriggeredChange = useCallback(
    (value: lineplot.TriggeredCustomRange) =>
      dispatch(lineplot.setCustomRange({ custom: value })),
    [dispatch],
  );
  const variant: Variant = custom?.variant === "triggered" ? "triggered" : "dynamic";
  return (
    <Flex.Box x wrap align="center">
      {FLAGS.lineplotWindows && (
        <Input.Item x label="Window">
          <SelectVariant value={variant} onChange={handleVariantChange} />
        </Input.Item>
      )}
      {custom?.variant === "triggered" ? (
        <TriggeredForm value={custom} onChange={handleTriggeredChange} />
      ) : (
        <Input.Item x label="Custom">
          <SpanInput
            value={custom?.variant === "dynamic" ? custom.span : 0}
            onChange={handleSpanChange}
            placeholder="1h 30m"
            grow
          />
        </Input.Item>
      )}
    </Flex.Box>
  );
};
