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

import { Input } from "./Input";
import { InputBaseProps } from "./types";

export interface InputDateProps extends InputBaseProps<number> {}

export const InputDate = forwardRef<HTMLInputElement, InputDateProps>(
  ({ size = "medium", onChange, value, ...props }, ref) => {
    const ts = new TimeStamp(value, "UTC");
    useEffect(() => {
      const rem = ts.remainder(TimeStamp.DAY);
      if (ts.remainder(TimeStamp.DAY).after(0)) 
        onChange(ts.sub(rem).valueOf());
    })
    return (
      <Input
        ref={ref}
        value={new TimeStamp(value, "UTC").fString("ISODate", "UTC")}
        onChange={(v) => {
          if (v.length === 0) return;
          onChange(new TimeStamp(v, "local").valueOf());
        }}
        type="date"
        {...props}
      />
    )
  }
);
InputDate.displayName = "InputDate";
