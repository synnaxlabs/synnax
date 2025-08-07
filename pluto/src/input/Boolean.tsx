// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { type InputProps } from "@/input/types";
import { stopPropagation } from "@/util/event";

export interface BooleanProps
  extends InputProps<boolean>,
    Omit<Button.ExtensionProps, "variant"> {
  inputType: "switch" | "checkbox";
}

const parseTextColor = (
  variant: Button.Variant,
  textColor: Button.ButtonProps["color"],
  value: boolean,
): Button.ButtonProps["color"] => {
  if (variant === "preview" && value === true) return "var(--pluto-primary-z)";
  return textColor;
};

/**
 * Base Boolean input component for switch and checkbox variants.
 */
export const Boolean = ({
  ref,
  className,
  value,
  disabled,
  onChange,
  inputType,
  size,
  variant = "text",
  style,
  color,
  borderColor,
  borderWidth,
  bordered,
  rounded,
  background,
  textColor,
  ...rest
}: BooleanProps): ReactElement => (
  <Button.Button
    el="label"
    variant={variant}
    className={CSS(CSS.BE("input", inputType), className)}
    disabled={disabled}
    size={size}
    preventClick
    style={style}
    color={color}
    borderColor={borderColor}
    borderWidth={borderWidth}
    bordered={bordered}
    rounded={rounded}
    background={background}
    textColor={parseTextColor(variant, textColor, value)}
  >
    {variant !== "preview" ? (
      <>
        <input
          className={CSS.BE("input", inputType, "input")}
          type="checkbox"
          ref={ref}
          checked={value}
          onChange={(e) => onChange?.(e.target.checked)}
          onClick={stopPropagation}
          value=""
          disabled={disabled}
          {...rest}
        />
        <span
          className={CSS.BE("input", inputType, "indicator")}
          onClick={stopPropagation}
        />
      </>
    ) : value ? (
      "True"
    ) : (
      "False"
    )}
  </Button.Button>
);
