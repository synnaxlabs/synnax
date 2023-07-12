// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentPropsWithoutRef, ReactElement } from "react";

import { Optional } from "@synnaxlabs/x";

import { CSS } from "@/core/css";
import { SpaceProps } from "@/core/std/Space";
import { Tooltip } from "@/core/std/Tooltip";
import { Typography, Text, TextWithIconProps } from "@/core/std/Typography";
import { ComponentSize } from "@/util/component";

import "@/core/std/Button/Button.css";

/** The variant of button */
export type ButtonVariant = "filled" | "outlined" | "text";

export interface ButtonExtensionProps {
  variant?: ButtonVariant;
  size?: ComponentSize;
  sharp?: boolean;
}

/** The base props accepted by all button types in this directory. */
export interface BaseButtonProps
  extends ComponentPropsWithoutRef<"button">,
    ButtonExtensionProps {}

/** The props for the {@link Button} component. */
export interface ButtonProps
  extends Optional<Omit<TextWithIconProps<"button">, "size">, "level">,
    ButtonExtensionProps {
  children?: string | number;
  startIcon?: ReactElement | ReactElement[];
  endIcon?: ReactElement | ReactElement[];
  iconSpacing?: SpaceProps["size"];
}

export const Button = Tooltip.wrap(
  ({
    size = "medium",
    variant = "filled",
    className,
    children,
    iconSpacing,
    sharp = false,
    level,
    ...props
  }: ButtonProps): ReactElement => (
    <Text.WithIcon
      el="button"
      className={CSS(
        CSS.B("btn"),
        CSS.size(size),
        CSS.sharp(sharp),
        CSS.BM("btn", variant),
        className
      )}
      level={level ?? Typography.ComponentSizeLevels[size]}
      size={iconSpacing}
      noWrap
      {...props}
    >
      {children}
    </Text.WithIcon>
  )
);
