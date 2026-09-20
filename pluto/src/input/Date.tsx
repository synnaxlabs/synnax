// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Date.css";

import { TimeStamp, type xy } from "@synnaxlabs/x";
import { type ReactElement, useLayoutEffect } from "react";

import { CSS } from "@/css";
import { DragButton } from "@/input/DragButton";
import { Text, type TextProps } from "@/input/Text";
import { type Control } from "@/input/types";

/** Props for {@link Date}. The value is UTC nanoseconds since the Unix epoch. */
export interface DateProps
  extends Omit<TextProps, "type" | "value" | "onChange">, Control<number> {
  showDragHandle?: boolean;
}

const DRAG_SCALE: xy.XY = {
  x: Number(TimeStamp.HOUR.valueOf()),
  y: Number(TimeStamp.days(0.75).valueOf()),
};

export interface UseDateProps extends Pick<DateProps, "value" | "onChange"> {}

export interface UseDateReturn {
  value: string;
  onChange: (value: string | number) => void;
}

/**
 * Backs a date field. It shows and accepts local time while the controlled value stays
 * UTC nanoseconds, and snaps the value to local midnight.
 */
export const useDate = ({ value, onChange }: UseDateProps): UseDateReturn => {
  const ts = new TimeStamp(value, "UTC");

  useLayoutEffect(() => {
    // We want the date to be at midnight in local time.
    const local = ts.sub(TimeStamp.utcOffset);
    if (local.remainder(TimeStamp.DAY).isZero) return;
    const tsV = local.sub(local.remainder(TimeStamp.DAY));
    // We have a correcly zeroed timestamp in local, now
    // add back the UTC offset to get the UTC timestamp.
    onChange(Number(new TimeStamp(tsV, "local").valueOf()));
  }, [value]);

  const handleChange = (value: string | number): void => {
    let ts: TimeStamp;
    // This is coming from the drag button. We give the drag button a value in UTC, and
    // it adds or subtracts a fixed amount of time, giving us a new UTC timestamp.
    if (typeof value === "number") ts = new TimeStamp(value, "UTC");
    // This means the user hasn't finished inputting a date.
    else if (value.length === 0) return;
    // No need to worry about taking remainders here. The input will prevent values over
    // a day. We interpret the input as local, which adds the UTC offset back in.
    else ts = new TimeStamp(value, "local");
    onChange(Number(ts.valueOf()));
  };

  // The props value is in UTC, but we want the user to view AND enter in local. This
  // subtracts the UTC offset from the timestamp.
  const inputValue = ts.toString("ISODate", "local");

  return { value: inputValue, onChange: handleChange };
};

/**
 * A date field over UTC nanoseconds. It shows local time and a drag handle that scrubs
 * by the hour horizontally and by the day vertically.
 */
export const Date = ({
  ref,
  onChange,
  value,
  className,
  showDragHandle = true,
  children,
  preview,
  ...rest
}: DateProps): ReactElement => {
  const { value: inputValue, onChange: handleChange } = useDate({ value, onChange });
  if (preview === true) showDragHandle = false;
  return (
    <Text
      ref={ref}
      value={inputValue}
      className={CSS.cls(CSS.B("input-date"), className)}
      onChange={handleChange}
      type="date"
      preview={preview}
      {...rest}
    >
      {showDragHandle && (
        <DragButton value={value} onChange={handleChange} dragScale={DRAG_SCALE} />
      )}
      {children}
    </Text>
  );
};
