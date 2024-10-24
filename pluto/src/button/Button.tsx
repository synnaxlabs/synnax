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
import {
  type ComponentPropsWithoutRef,
  type ReactElement,
  ReactNode,
  useCallback,
  useRef,
} from "react";

import { type Align } from "@/align";
import { Color } from "@/color";
import { CSS } from "@/css";
import { status } from "@/status/aether";
import { Text } from "@/text";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";
import { type ComponentSize } from "@/util/component";

/** The variant of button */
export type Variant =
  | "filled"
  | "outlined"
  | "text"
  | "suggestion"
  | "preview"
  | "shadow";

export interface ButtonExtensionProps {
  variant?: Variant;
  size?: ComponentSize;
  sharp?: boolean;
  loading?: boolean;
  triggers?: Triggers.Trigger[];
  status?: status.Variant;
  color?: Color.Crude;
}

/** The base props accepted by all button types in this directory. */
export interface BaseProps
  extends Omit<ComponentPropsWithoutRef<"button">, "color">,
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
    endContent?: ReactNode;
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
    size,
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
    style,
    endContent,
    onMouseDown,
    ...props
  }: ButtonProps): ReactElement => {
    const parsedDelay = TimeSpan.fromMilliseconds(delay);

    if (loading) startIcon = [...toArray(startIcon), <Icon.Loading key="loader" />];
    if (iconSpacing == null) iconSpacing = size === "small" ? "small" : "medium";
    // We implement the shadow variant to maintain compatibility with the input
    // component API.
    if (variant == "shadow") variant = "text";

    const handleClick: ButtonProps["onClick"] = (e) => {
      if (disabled || variant === "preview") return;
      if (parsedDelay.isZero) return onClick?.(e);
    };

    const toRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    const handleMouseDown: ButtonProps["onMouseDown"] = (e) => {
      onMouseDown?.(e);
      if (disabled || variant === "preview" || parsedDelay.isZero) return;
      document.addEventListener(
        "mouseup",
        () => toRef.current != null && clearTimeout(toRef.current),
      );
      toRef.current = setTimeout(() => {
        onClick?.(e);
        toRef.current = null;
      }, parsedDelay.milliseconds);
    };

    Triggers.use({
      triggers,
      callback: useCallback<(e: Triggers.UseEvent) => void>(
        ({ stage }) => {
          if (stage !== "end" || disabled || variant === "preview") return;
          handleClick(
            new MouseEvent("click") as unknown as React.MouseEvent<HTMLButtonElement>,
          );
        },
        [handleClick, disabled],
      ),
    });

    const pStyle = { ...style };
    const res = Color.Color.z.safeParse(color);
    const hasCustomColor =
      res.success && (variant === "filled" || variant === "outlined");
    if (hasCustomColor) {
      // @ts-expect-error - css variable
      pStyle[CSS.var("btn-color")] = res.data.rgbString;
      // @ts-expect-error - css variable
      pStyle[CSS.var("btn-text-color")] = res.data.pickByContrast(
        "#000000",
        "#ffffff",
      ).rgbCSS;
    }

    if (!parsedDelay.isZero)
      // @ts-expect-error - css variable
      pStyle[CSS.var("btn-delay")] = `${parsedDelay.seconds.toString()}s`;

    if (size == null && level != null) size = Text.LevelComponentSizes[level];
    else if (size != null && level == null) level = Text.ComponentSizeLevels[size];
    else if (size == null) size = "medium";

    return (
      <Text.WithIcon<"button", any>
        el="button"
        className={CSS(
          CSS.B("btn"),
          CSS.size(size),
          CSS.sharp(sharp),
          variant !== "preview" && CSS.disabled(disabled),
          status != null && CSS.M(status),
          CSS.BM("btn", variant),
          hasCustomColor && CSS.BM("btn", "custom-color"),
          className,
        )}
        type={type}
        level={level ?? Text.ComponentSizeLevels[size]}
        size={iconSpacing}
        onClick={handleClick}
        onMouseDown={handleMouseDown}
        noWrap
        style={pStyle}
        startIcon={startIcon}
        color={color}
        {...props}
      >
        {children}
        {endContent != null ? (
          <div className={CSS.BE("btn", "end-content")}>
            {Text.formatChildren(level ?? Text.ComponentSizeLevels[size], endContent)}
          </div>
        ) : undefined}
      </Text.WithIcon>
    );
  },
);
