// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Time.css";

import { type direction, TimeSpan, TimeStamp, type TZInfo } from "@synnaxlabs/x";
import { useCallback } from "react";

import { CSS } from "@/css";
import { DragButton } from "@/input/DragButton";
import { Text, type TextProps } from "@/input/Text";
import { type Control } from "@/input/types";

export const combineDateAndTimeValue = (date: number, time: number): TimeStamp =>
  new TimeStamp(date).add(time).sub(TimeStamp.utcOffset);

export interface TimeProps
  extends Omit<TextProps, "type" | "value" | "onChange">,
    Control<number> {
  tzInfo?: TZInfo;
  showDragHandle?: boolean;
  dragDirection?: direction.Direction;
}

const DRAG_SCALE = {
  x: Number(TimeSpan.SECOND.valueOf()) * 0.5,
  y: Number(TimeSpan.MINUTE.valueOf()),
};

interface UseTimeProps extends Pick<TimeProps, "value" | "onChange" | "tzInfo"> {}

export interface UseTimeReturn {
  inputValue: string;
  ts: TimeStamp;
  handleChange: Control<string | number>["onChange"];
}

export const useTime = ({ value, onChange, tzInfo }: UseTimeProps): UseTimeReturn => {
  const ts = new TimeStamp(value, "UTC");

  // We want to check for remainder overflow in LOCAL time.
  const local = ts.sub(TimeStamp.utcOffset);
  // All good.
  if (local.after(TimeStamp.DAY)) {
    // Chop off the extra time.
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
    [onChange, tzInfo],
  );

  const inputValue = ts.toString("time", tzInfo);

  return { inputValue, ts, handleChange };
};

/**
 * A controlled Time input component.
 *
 * @param props - The props for the input component. Unlisted props are passed to the
 * underlying input element.
 * @param props.value - The value of the input.
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
 * @default x: 1/2 Second, y: 1 Minute
 * @param props.dragDirection - The direction of the drag handle.
 * @default undefined
 */
export const Time = ({
  ref,
  size,
  value,
  tzInfo = "local",
  onChange,
  dragDirection,
  showDragHandle = true,
  className,
  children,
  ...rest
}: TimeProps) => {
  const { inputValue, ts, handleChange } = useTime({ value, onChange, tzInfo });
  return (
    <Text
      ref={ref}
      value={inputValue}
      className={CSS(CSS.B("input-time"), className)}
      type="time"
      step="1"
      onChange={handleChange}
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
