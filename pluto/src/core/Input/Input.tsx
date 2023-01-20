// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef } from "react";

import clsx from "clsx";

import { InputBaseProps } from "./types";

import "./Input.css";

export interface InputProps extends InputBaseProps<string> {
  selectOnFocus?: boolean;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  (
    {
      size = "medium",
      value,
      onChange,
      className,
      onFocus,
      selectOnFocus = false,
      ...props
    },
    ref
  ) => (
    <input
      ref={ref}
      value={value ?? ""}
      className={clsx(
        "pluto-input",
        `pluto--${size}`,
        `pluto-input--${size}`,
        className
      )}
      onChange={(e) => onChange(e.target.value)}
      onFocus={(e) => {
        if (selectOnFocus) e.target.select();
        onFocus?.(e);
      }}
      {...props}
    />
  )
);
Input.displayName = "Input";
