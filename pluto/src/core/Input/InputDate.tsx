// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, useCallback, useEffect } from "react";

import { TimeStamp } from "@synnaxlabs/x";
import clsx from "clsx";

import { Pack } from "../Pack";

import { Input } from "./Input";
import { InputDragButton, InputDragButtonExtensionProps } from "./InputDragButton";
import { InputBaseProps } from "./types";

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
      const rem = ts.remainder(TimeStamp.DAY);
      if (ts.remainder(TimeStamp.DAY).after(0)) onChange(ts.sub(rem).valueOf());
    });

    const handleChange = useCallback((value: number | string) => {
      if (typeof value === "number") {
        const ts = new TimeStamp(value);
        onChange(ts.sub(ts.remainder(TimeStamp.DAY)).valueOf());
      } else if (value.length > 0) {
        onChange(new TimeStamp(value, "local").remainder(TimeStamp.DAY).valueOf());
      }
    }, []);

    const input = (
      <Input
        ref={ref}
        value={new TimeStamp(value, "UTC").fString("ISODate", "UTC")}
        className={clsx("pluto-input-date", className)}
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
