// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import clsx from "clsx";

import { SpaceProps } from "@/core/Space";
import { Typography, Text } from "@/core/Typography";
import { ComponentSize } from "@/util/component";

import "./Button.css";

/** The variant of button */
export type ButtonVariant = "filled" | "outlined" | "text";

/** The base props accepted by all button types in this directory. */
export interface BaseButtonProps extends Omit<SpaceProps<"button">, "el" | "value"> {
  variant?: ButtonVariant;
  size?: ComponentSize;
}

/** The props for the {@link Button} component. */
export interface ButtonProps extends BaseButtonProps {
  children?: string | number;
  startIcon?: ReactElement | ReactElement[];
  endIcon?: ReactElement | ReactElement[];
  iconSpacing?: SpaceProps["size"];
}

export const Button = ({
  size = "medium",
  variant = "filled",
  className,
  children,
  iconSpacing,
  ...props
}: ButtonProps): JSX.Element => (
  <Text.WithIcon
    el="button"
    className={clsx("pluto--" + size, "pluto-btn", "pluto-btn--" + variant, className)}
    color={variant === "filled" ? "var(--pluto-white)" : "var(--pluto-text-color)"}
    level={Typography.ComponentSizeLevels[size]}
    size={iconSpacing ?? size}
    {...props}
  >
    {children}
  </Text.WithIcon>
);
