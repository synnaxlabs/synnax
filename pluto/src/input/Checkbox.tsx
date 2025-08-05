// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Checkbox.css";

import { type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { type InputProps } from "@/input/types";
import { preventDefault, stopPropagation } from "@/util/event";

export interface CheckboxProps
  extends InputProps<boolean>,
    Omit<Button.ExtensionProps, "variant"> {}

/**
 * A controlled boolean Checkbox input component.
 *
 * @param props - The props for the input component. Unlisted props are passed to the
 * underlying input element.
 * @param props.value - The value of the input.
 * @param props.onChange - A function to call when the input value changes.
 * @param props.size - The size of the input: "small" | "medium" | "large".
 * @default "medium"
 */
export const Checkbox = ({
  ref,
  className,
  value,
  disabled,
  onChange,
  size = "medium",
  variant,
  color,
  borderColor,
  borderWidth,
  bordered,
  rounded,
  background,
  ...rest
}: CheckboxProps): ReactElement => (
  <Button.Button
    el="label"
    variant="text"
    className={CSS(CSS.BE("input", "checkbox"), className)}
    size={size}
    preventClick
    {...(rest as Button.ButtonProps<"label">)}
  >
    <input
      className={CSS.BE("input", "checkbox", "input")}
      type="checkbox"
      ref={ref}
      checked={value}
      onMouseDown={preventDefault}
      onChange={(e) => onChange?.(e.target.checked)}
      disabled={disabled}
      onClick={stopPropagation}
    />
    <span
      className={CSS.BE("input", "checkbox", "checkmark")}
      onClick={stopPropagation}
    />
  </Button.Button>
);
