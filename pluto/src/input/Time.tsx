// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Time.css";

import { type direction, TimeSpan, TimeStamp, type TimeZone } from "@synnaxlabs/x";
import { useCallback } from "react";

import { CSS } from "@/css";
import { DragButton } from "@/input/DragButton";
import { Text, type TextProps } from "@/input/Text";
import { type Control } from "@/input/types";

/** Joins a {@link Date} value and a {@link Time} value into one UTC timestamp. */
export const combineDateAndTimeValue = (date: number, time: number): TimeStamp =>
  new TimeStamp(date).add(time).sub(TimeStamp.utcOffset);

/** Props for {@link Time}. The value is UTC nanoseconds past midnight. */
export interface TimeProps
  extends Omit<TextProps, "type" | "value" | "onChange">, Control<number> {
  /** Zone the value is shown and entered in. Defaults to local. */
  timeZone?: TimeZone;
  showDragHandle?: boolean;
  dragDirection?: direction.Direction;
}

const DRAG_SCALE = {
  x: Number(TimeSpan.SECOND.valueOf()) * 0.5,
  y: Number(TimeSpan.MINUTE.valueOf()),
};

interface UseTimeProps extends Pick<TimeProps, "value" | "onChange" | "timeZone"> {}

export interface UseTimeReturn {
  inputValue: string;
  ts: TimeStamp;
  handleChange: Control<string | number>["onChange"];
}

/**
 * Backs a time field. It shows and accepts the given zone while the controlled value
 * stays UTC nanoseconds, and wraps a value past a full day back into range.
 */
export const useTime = ({ value, onChange, timeZone }: UseTimeProps): UseTimeReturn => {
  const ts = new TimeStamp(value, "UTC");

  // We want to check for remainder overflow in LOCAL time.
  const local = ts.sub(TimeStamp.utcOffset);
  if (local.after(TimeStamp.DAY)) {
    const tsV = local.remainder(TimeStamp.DAY);
    // We have a correcly zeroed timestamp in local, now
    // add back the UTC offset to get the UTC timestamp.
    onChange(Number(new TimeStamp(tsV, "local").valueOf()));
  }

  const handleChange = useCallback(
    (value: number | string) => {
      let ts: TimeStamp;
      if (typeof value === "number") ts = new TimeStamp(value, "UTC");
      else if (value.length === 0) return;
      else ts = new TimeStamp(value, "local");
      onChange(Number(ts.valueOf()));
    },
    [onChange, timeZone],
  );

  const inputValue = ts.toString("time", timeZone);

  return { inputValue, ts, handleChange };
};

/**
 * A time-of-day field over UTC nanoseconds. It shows local time and a drag handle that
 * scrubs by the half second horizontally and by the minute vertically.
 */
export const Time = ({
  ref,
  size,
  value,
  timeZone = "local",
  onChange,
  dragDirection,
  showDragHandle = true,
  className,
  children,
  preview,
  ...rest
}: TimeProps) => {
  const { inputValue, ts, handleChange } = useTime({ value, onChange, timeZone });
  if (preview === true) showDragHandle = false;
  return (
    <Text
      ref={ref}
      value={inputValue}
      className={CSS.cls(CSS.B("input-time"), className)}
      type="time"
      step="1"
      onChange={handleChange}
      preview={preview}
      {...rest}
    >
      {showDragHandle && (
        <DragButton
          direction={dragDirection}
          value={Number(ts.valueOf())}
          onChange={handleChange}
          dragScale={DRAG_SCALE}
        />
      )}
      {children}
    </Text>
  );
};
