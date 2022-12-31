// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef } from "react";

import { InputProps } from "./Input";

export interface InputDateProps extends InputProps {}

export const InputDate = forwardRef<HTMLInputElement, InputDateProps>(
  ({ size = "medium", ...props }, ref) => {
    return (
      <input
        ref={ref}
        type="date"
        className={`pluto-input__input pluto-input--${size}`}
        {...props}
      />
    );
  }
);
InputDate.displayName = "InputDate";
