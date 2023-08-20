// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef } from "react";

import { CSS } from "@/css";
import { BaseProps } from "@/input/types";

import "@/input/Input.css";

export interface TextProps extends BaseProps<string> {
  selectOnFocus?: boolean;
  centerPlaceholder?: boolean;
}

/**
 * A controlled string input component.
 *
 * @param props - The props for the input component. Unlisted props are passed to the
 * underlying input element.
 * @param props.value - The value of the input.
 * @param props.onChange - A function to call when the input value changes.
 * @param props.size - The size of the input: "small" | "medium" | "large".
 * @param props.selectOnFocus - Whether the input should select its contents when focused.
 * @param props.centerPlaceholder - Whether the placeholder should be centered.
 */
export const Text = forwardRef<HTMLInputElement, TextProps>(
  (
    {
      size = "medium",
      value,
      onChange,
      className,
      onFocus,
      selectOnFocus = false,
      centerPlaceholder = false,
      variant = "outlined",
      ...props
    },
    ref
  ) => (
    <input
      ref={ref}
      value={value}
      className={CSS(
        CSS.B("input"),
        CSS.size(size),
        CSS.BM("input", variant),
        centerPlaceholder && CSS.BM("input", "placeholder-centered"),
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
Text.displayName = "Input";
