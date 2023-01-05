// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef } from "react";

import { Input } from "./Input";
import { InputBaseProps } from "./types";

import { nanoTimeString, timeStringNano } from "@/util/time";

export interface InputTimeProps extends InputBaseProps<number> {}

export const InputTime = forwardRef<HTMLInputElement, InputTimeProps>(
  ({ size = "medium", value, onChange, ...props }: InputTimeProps, ref) => {
    return (
      <Input
        ref={ref}
        type="time"
        step="1"
        value={nanoTimeString(value)}
        onChange={(value) => value.length > 0 && onChange(timeStringNano(value))}
        {...props}
      />
    );
  }
);
InputTime.displayName = "InputTime";
