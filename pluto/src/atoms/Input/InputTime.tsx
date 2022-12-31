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

export interface InputTimeProps extends InputProps {}

export const InputTime = forwardRef<HTMLInputElement, InputTimeProps>(
  ({ size = "medium", ...props }: InputTimeProps, ref) => {
    return (
      <input
        ref={ref}
        type="time"
        step="1"
        className={`pluto-input__input pluto-input--${size}`}
        {...props}
      />
    );
  }
);
InputTime.displayName = "InputTime";
