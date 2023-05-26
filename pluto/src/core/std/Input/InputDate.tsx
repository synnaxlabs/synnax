// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, useEffect } from "react";

import { TimeStamp } from "@synnaxlabs/x";

import { Input } from "@/core/std/Input";
import {
  InputDragButton,
  InputDragButtonExtensionProps,
} from "@/core/std/Input/InputDragButton";
import { InputBaseProps } from "@/core/std/Input/types";
import { Pack } from "@/core/std/Pack";
import { CSS } from "@/core/css";

import "./InputDate.css";

export interface InputDateProps
  extends InputBaseProps<number>,
    InputDragButtonExtensionProps {
  showDragHandle?: boolean;
}

const DRAG_SCALE = {
  x: TimeStamp.HOUR.valueOf(),
  y: TimeStamp.DAY.valueOf() * 0.75,
};

export const InputDate = forwardRef<HTMLInputElement, InputDateProps>(
  (
    { size = "medium", onChange, value, className, showDragHandle = true, ...props },
    ref
  ) => {
    const ts = new TimeStamp(value, "UTC");

    useEffect(() => {
      // We want the date to be at midnight in local time.
      const local = ts.sub(TimeStamp.utcOffset);
      // All good.
      if (local.remainder(TimeStamp.DAY).isZero) return;
      // If it isn't, take off the extra time.
      const tsV = local.sub(local.remainder(TimeStamp.DAY));
      // We have a correcly zeroed timestamp in local, now
      // add back the UTC offset to get the UTC timestamp.
      onChange(new TimeStamp(tsV, "local").valueOf());
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
      onChange(ts.valueOf());
    };

    // The props value is in UTC, but we want the user
    // to view AND enter in local. This subtracts the
    // UTC offset from the timestamp.
    const inputValue = ts.fString("ISODate", "local");
    const input = (
      <Input
        ref={ref}
        value={inputValue}
        className={CSS(CSS.B("input-date"), className)}
        onChange={handleChange}
        type="date"
        {...props}
      />
    );

    if (!showDragHandle) return input;
    return (
      <Pack>
        {input}
        <InputDragButton value={value} onChange={handleChange} dragScale={DRAG_SCALE} />
      </Pack>
    );
  }
);
InputDate.displayName = "InputDate";
