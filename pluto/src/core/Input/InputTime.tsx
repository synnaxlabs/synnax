// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, useEffect } from "react";

import { TimeStamp, TZInfo } from "@synnaxlabs/x";

import { Input } from "./Input";
import { InputBaseProps } from "./types";

export interface InputTimeProps extends InputBaseProps<number> {
  tzInfo?: TZInfo;
}

export const InputTime = forwardRef<HTMLInputElement, InputTimeProps>(
  (
    { size = "medium", value, tzInfo = "local", onChange, ...props }: InputTimeProps,
    ref
  ) => {
    const ts = new TimeStamp(value, "UTC");
    useEffect(() => {
      if (!ts.after(TimeStamp.DAY)) return;
      const tsV = ts.remainder(TimeStamp.DAY);
      onChange(tsV.valueOf());
    });
    return (
      <Input
        ref={ref}
        type="time"
        step="1"
        value={ts.fString("time", tzInfo)}
        onChange={(value) =>
          value.length > 0 && onChange(new TimeStamp(value, tzInfo).valueOf())
        }
        {...props}
      />
    );
  }
);
InputTime.displayName = "InputTime";
