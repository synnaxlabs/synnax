// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/button/Button.css";

import { Icon } from "@synnaxlabs/media";
import { TimeSpan } from "@synnaxlabs/x/telem";
import { toArray } from "@synnaxlabs/x/toArray";
import { type ComponentPropsWithoutRef, type ReactElement } from "react";

import { type Align } from "@/align";
import { Color } from "@/color";
import { CSS } from "@/css";
import { status } from "@/status/aether";
import { Text } from "@/text";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";
import { type ComponentSize } from "@/util/component";

/** The variant of button */
export type Variant = "filled" | "outlined" | "text" | "suggestion";

export interface ButtonExtensionProps {
  variant?: Variant;
  size?: ComponentSize;
  sharp?: boolean;
  loading?: boolean;
  triggers?: Triggers.Trigger[];
  status?: status.Variant;
}

/** The base props accepted by all button types in this directory. */
export interface BaseProps
  extends ComponentPropsWithoutRef<"button">,
    ButtonExtensionProps {}

/** The props for the {@link Button} component. */
export type ButtonProps = Omit<
  Text.WithIconProps<"button">,
  "size" | "startIcon" | "endIcon" | "level"
> &
  ButtonExtensionProps &
  BaseProps & {
    level?: Text.Level;
    startIcon?: ReactElement | ReactElement[];
    endIcon?: ReactElement | ReactElement[];
    iconSpacing?: Align.SpaceProps["size"];
    disabled?: boolean;
    delay?: number | TimeSpan;
  };

/**
 * Use is a basic button component.
 *
 * @param props - Props for the component, which are passed down to the underlying button
 * element.
 * @param props.size - The size of button render.
 * @param props.variant - The variant to render for the button. Options are "filled"
 * (default), "outlined", and "text".
 * @param props.startIcon - An optional icon to render before the start of the button
 * text. This can be a single icon or an array of icons. The icons will be formatted
 * to match the color and size of the button.
 * @param props.endIcon - The same as {@link startIcon}, but renders after the button
 * text.
 */
export const Button = Tooltip.wrap(
  ({
    size = "medium",
    variant = "filled",
    type = "button",
    className,
    children,
    iconSpacing,
    sharp = false,
    disabled = false,
    loading = false,
    level,
    triggers,
    startIcon = [] as ReactElement[],
    delay = 0,
    onClick,
    color,
    status,
    ...props
  }: ButtonProps): ReactElement => {
    if (loading) startIcon = [...toArray(startIcon), <Icon.Loading key="loader" />];
    if (iconSpacing == null) iconSpacing = size === "small" ? "small" : "medium";

    const handleClick: ButtonProps["onClick"] = (e) => {
      if (disabled) return;
      const span = delay instanceof TimeSpan ? delay : TimeSpan.milliseconds(delay);
      if (span.isZero) return onClick?.(e);
    };

    Triggers.use({
      triggers,
      // @ts-expect-error
      callback: ({ stage }) => stage === "end" && handleClick(new MouseEvent("click")),
    });

    return (
      <Text.WithIcon<"button", any>
        el="button"
        className={CSS(
          CSS.B("btn"),
          CSS.size(size),
          CSS.sharp(sharp),
          CSS.disabled(disabled),
          status != null && CSS.M(status),
          CSS.BM("btn", variant),
          className,
        )}
        type={type}
        level={level ?? Text.ComponentSizeLevels[size]}
        size={iconSpacing}
        onClick={handleClick}
        noWrap
        color={Color.cssString(color)}
        startIcon={startIcon}
        {...props}
      >
        {children}
      </Text.WithIcon>
    );
  },
);
