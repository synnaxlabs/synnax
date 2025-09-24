// Copyright 2025 Synnax Labs, Inc.
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

export interface DateProps
  extends Omit<TextProps, "type" | "value" | "onChange">,
    Control<number> {
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

export const useDate = ({ value, onChange }: UseDateProps): UseDateReturn => {
  const ts = new TimeStamp(value, "UTC");

  useLayoutEffect(() => {
    // We want the date to be at midnight in local time.
    const local = ts.sub(TimeStamp.utcOffset);
    // All good.
    if (local.remainder(TimeStamp.DAY).isZero) return;
    // If it isn't, take off the extra time.
    const tsV = local.sub(local.remainder(TimeStamp.DAY));
    // We have a correcly zeroed timestamp in local, now
    // add back the UTC offset to get the UTC timestamp.
    onChange(Number(new TimeStamp(tsV, "local").valueOf()));
  }, [value]);

  const handleChange = (value: string | number): void => {
    let ts: TimeStamp;
    // This is coming from the drag button. We give the drag
    // button a value in UTC, and it adds or subtracts a fixed
    // amount of time, giving us a new UTC timestamp.
    if (typeof value === "number") ts = new TimeStamp(value, "UTC");
    // This means the user hasn't finished inputting a date.
    else if (value.length === 0) return;
    // No need to worry about taking remainders here. The input
    // will prevent values over a day. We interpret the input as
    // local, which adds the UTC offset back in.
    else ts = new TimeStamp(value, "local");
    onChange(Number(ts.valueOf()));
  };

  // The props value is in UTC, but we want the user
  // to view AND enter in local. This subtracts the
  // UTC offset from the timestamp.
  const inputValue = ts.toString("ISODate", "local");

  return { value: inputValue, onChange: handleChange };
};

/**
 * A controlled Date input component.
 *
 * @param props - The props for the input component. Unlisted props are passed to the
 * underlying input element.
 * @param props.value - The value of the input represented as a number of nanoseconds
 * since the Unix epoch.
 * @param props.onChange - A function to call when the input value changes.
 * @param props.size - The size of the input: "small" | "medium" | "large".
 * @default "medium"
 * @param props.selectOnFocus - Whether the input should select its contents when focused.
 * @defaul true
 * @param props.centerPlaceholder - Whether the placeholder should be centered.
 * @default false
 * @param props.showDragHandle - Whether or not to show a drag handle to set the time.
 * @default true
 * @param props.dragScale - The scale of the drag handle.
 * @default x: 1 Hour, y: 3/4 Day
 * @param props.dragDirection - The direction of the drag handle.
 * @default undefined
 */
export const Date = ({
  ref,
  onChange,
  value,
  className,
  showDragHandle = true,
  children,
  ...rest
}: DateProps): ReactElement => {
  const { value: inputValue, onChange: handleChange } = useDate({ value, onChange });
  return (
    <Text
      ref={ref}
      value={inputValue}
      className={CSS(CSS.B("input-date"), className)}
      onChange={handleChange}
      type="date"
      {...rest}
    >
      {showDragHandle && (
        <DragButton value={value} onChange={handleChange} dragScale={DRAG_SCALE} />
      )}
      {children}
    </Text>
  );
};
