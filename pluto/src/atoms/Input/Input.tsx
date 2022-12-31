// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DetailedHTMLProps,
  InputHTMLAttributes,
  RefAttributes,
  forwardRef,
} from "react";

import "./Input.css";
import clsx from "clsx";

import { ComponentSize } from "@/util";

export interface BaseInputProps
  extends Omit<
    DetailedHTMLProps<InputHTMLAttributes<HTMLInputElement>, HTMLInputElement>,
    "size" | "ref"
  > {}

export interface InputProps extends BaseInputProps, RefAttributes<HTMLInputElement> {
  size?: ComponentSize;
  name?: string;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ size = "medium", placeholder, value, className, ...props }, ref) => {
    return (
      <input
        ref={ref}
        placeholder={placeholder}
        className={clsx("pluto-input__input", "pluto-input--" + size, className)}
        {...props}
        value={value}
      />
    );
  }
);
Input.displayName = "Input";
