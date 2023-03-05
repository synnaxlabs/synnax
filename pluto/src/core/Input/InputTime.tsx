// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, useCallback, useEffect } from "react";

import { TimeSpan, TimeStamp, TZInfo } from "@synnaxlabs/x";

import { Pack } from "../Pack";

import { Input } from "./Input";
import { InputDragButton, InputDragButtonExtensionProps } from "./InputDragButton";
import { InputNumberProps } from "./InputNumber";
import { InputBaseProps } from "./types";

import { CSS } from "@/css";

import "./InputTime.css";

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
    useEffect(() => {
      if (!ts.after(TimeStamp.DAY.add(TimeStamp.utcOffset))) return;
      const tsV = new TimeStamp(value, "UTC").remainder(TimeStamp.DAY);
      onChange(tsV.valueOf());
    });

    const handleChange = useCallback(
      (value: number | string) => {
        if (typeof value === "number") {
          const ts = new TimeStamp(value, "UTC");
          onChange(ts.valueOf());
        } else if (value.length > 0) {
          onChange(new TimeStamp(value, tzInfo).valueOf());
        }
      },
      [onChange, tzInfo]
    );

    const input = (
      <Input
        ref={ref}
        className={CSS(CSS.B("input-time"), className)}
        type="time"
        step="1"
        value={ts.fString("time", tzInfo)}
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
