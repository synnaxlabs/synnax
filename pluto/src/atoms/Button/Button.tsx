// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ButtonHTMLAttributes, ReactElement } from "react";

import "./Button.css";
import clsx from "clsx";

import { ComponentSizeTypographyLevels, Text } from "@/atoms/Typography";
import { ComponentSize } from "@/util";

/** The base props accepted by all button types in this directory. */
export interface BaseButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "filled" | "outlined" | "text";
  size?: ComponentSize;
}

/** The props for the {@link Button} component. */
export interface ButtonProps extends BaseButtonProps {
  children: string | number;
  startIcon?: ReactElement | ReactElement[];
  endIcon?: ReactElement | ReactElement[];
}

export const Button = ({
  size = "medium",
  variant = "filled",
  className,
  startIcon,
  endIcon,
  children,
  ...props
}: ButtonProps): JSX.Element => {
  return (
    <button
      className={clsx(
        "pluto-btn",
        "pluto-btn--" + variant,
        "pluto-btn--" + size,
        className
      )}
      {...props}
    >
      <Text.WithIcon
        color={variant === "filled" ? "var(--pluto-white)" : "var(--pluto-text-color)"}
        level={ComponentSizeTypographyLevels[size]}
        startIcon={startIcon}
        endIcon={endIcon}
      >
        {children}
      </Text.WithIcon>
    </button>
  );
};
