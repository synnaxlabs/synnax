// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, useCallback } from "react";

import { TimeSpan, TimeStamp, TZInfo } from "@synnaxlabs/x";

import { CSS } from "@/core/css";
import { Input } from "@/core/std/Input/Input";
import {
  InputDragButton,
  InputDragButtonExtensionProps,
} from "@/core/std/Input/InputDragButton";
import { InputNumberProps } from "@/core/std/Input/InputNumber";
import { InputBaseProps } from "@/core/std/Input/types";
import { Pack } from "@/core/std/Pack";

import "@/core/std/Input/InputTime.css";

export interface InputTimeProps
  extends InputBaseProps<number>,
    InputDragButtonExtensionProps {
  tzInfo?: TZInfo;
  showDragHandle?: boolean;
}

const DRAG_SCALE = {
  x: TimeSpan.SECOND.valueOf() * 0.5,
  y: TimeSpan.MINUTE.valueOf(),
};

export const InputTime = forwardRef<HTMLInputElement, InputTimeProps>(
  (
    {
      size = "medium",
      value,
      tzInfo = "local",
      onChange,
      dragDirection,
      showDragHandle = true,
      className,
      ...props
    }: InputTimeProps,
    ref
  ) => {
    const ts = new TimeStamp(value, "UTC");

    // We want to check for remainder overflow in LOCAL time.
    const local = ts.sub(TimeStamp.utcOffset);
    // All good.
    if (local.after(TimeStamp.DAY)) {
      // Chop off the extra time.
      const tsV = local.remainder(TimeStamp.DAY);
      // We have a correcly zeroed timestamp in local, now
      // add back the UTC offset to get the UTC timestamp.
      onChange(new TimeStamp(tsV, "local").valueOf());
    }

    const handleChange = useCallback(
      (value: number | string) => {
        let ts: TimeStamp;
        if (typeof value === "number") ts = new TimeStamp(value, "UTC");
        else if (value.length === 0) return;
        else ts = new TimeStamp(value, "local");
        onChange(ts.valueOf());
      },
      [onChange, tzInfo]
    );

    const inputValue = ts.fString("time", tzInfo);
    const input = (
      <Input
        ref={ref}
        value={inputValue}
        className={CSS(CSS.B("input-time"), className)}
        type="time"
        step="1"
        onChange={handleChange as InputBaseProps["onChange"]}
        {...props}
      />
    );

    if (!showDragHandle) return input;
    return (
      <Pack {...props}>
        {input}
        <InputDragButton
          direction={dragDirection}
          value={ts.valueOf()}
          onChange={handleChange as InputNumberProps["onChange"]}
          dragScale={DRAG_SCALE}
        />
      </Pack>
    );
  }
);
InputTime.displayName = "InputTime";
