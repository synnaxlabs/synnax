// Copyright 2022 Synnax Labs, Inc.
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

import { isoStringShortDate, shortDateISOString } from "@/util/time";

export interface InputDateProps extends InputBaseProps<number> {}

export const InputDate = forwardRef<HTMLInputElement, InputDateProps>(
  ({ size = "medium", onChange, value, ...props }, ref) => (
    <Input
      ref={ref}
      value={shortDateISOString(value)}
      onChange={(v) => onChange(isoStringShortDate(v))}
      type="date"
      {...props}
    />
  )
);
InputDate.displayName = "InputDate";
