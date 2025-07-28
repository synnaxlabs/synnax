// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/button/Button.css";

import { color, type status } from "@synnaxlabs/x";
import { array } from "@synnaxlabs/x/array";
import { TimeSpan } from "@synnaxlabs/x/telem";
import {
  type ComponentPropsWithRef,
  type ReactElement,
  useCallback,
  useRef,
} from "react";

import { type Align } from "@/align";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";

/** The variant of button */
export type Variant =
  | "filled"
  | "outlined"
  | "text"
  | "suggestion"
  | "preview"
  | "shadow";

/** The base props accepted by all button types in this directory. */
export interface BaseProps extends Omit<ComponentPropsWithRef<"button">, "color"> {
  variant?: Variant;
  size?: Component.Size;
  sharp?: boolean;
  loading?: boolean;
  triggers?: Triggers.Trigger | Triggers.Trigger[];
  status?: status.Variant;
  color?: color.Crude;
  textShade?: Text.Shade;
}

/** The props for the {@link Button} component. */
export type ButtonProps = Omit<
  Text.WithIconProps<"button">,
  "size" | "startIcon" | "endIcon" | "level"
> &
  Tooltip.WrapProps &
  BaseProps & {
    level?: Text.Level;
    startIcon?: Icon.ReactElement | Icon.ReactElement[];
    endIcon?: Icon.ReactElement | Icon.ReactElement[];
    disabled?: boolean;
    onClickDelay?: number | TimeSpan;
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
 * @param props.iconSpacing - The spacing between the optional start and end icons
 * and the button text. Can be "small", "medium", "large", or a number representing
 * the spacing in rem.
 * @param props.onClickDelay - An optional delay to wait before calling the `onClick`
 * handler. This will cause the button to render a progress bar that fills up over the
 * specified time before calling the handler.
 * @param props.loading - Whether the button is in a loading state. This will cause the
 * button to render a loading spinner.
 */
export const Button = Tooltip.wrap(
  ({
    size,
    variant = "outlined",
    type = "button",
    className,
    children,
    gap,
    sharp = false,
    disabled = false,
    loading = false,
    level,
    triggers,
    startIcon = [],
    onClickDelay = 0,
    onClick,
    color: colorVal,
    status,
    style,
    onMouseDown,
    shade = 0,
    textShade,
    tabIndex,
    ...rest
  }: ButtonProps): ReactElement => {
    if (variant == "outlined" && shade == null) shade = 0;
    const parsedDelay = TimeSpan.fromMilliseconds(onClickDelay);
    if (loading)
      startIcon = [...array.toArray(startIcon), <Icon.Loading key="loader" />];
    const isDisabled = disabled || loading;
    gap ??= size === "small" ? "small" : "medium";
    // We implement the shadow variant to maintain compatibility with the input
    // component API.
    if (variant == "shadow") variant = "text";

    const handleClick: ButtonProps["onClick"] = (e) => {
      if (isDisabled || variant === "preview") return;
      if (parsedDelay.isZero) return onClick?.(e);
    };

    const toRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    const handleMouseDown: ButtonProps["onMouseDown"] = (e) => {
      if (tabIndex == -1) e.preventDefault();
      onMouseDown?.(e);
      if (isDisabled || variant === "preview" || parsedDelay.isZero) return;
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
          if (stage !== "end" || isDisabled || variant === "preview") return;
          handleClick(
            new MouseEvent("click") as unknown as React.MouseEvent<HTMLButtonElement>,
          );
        },
        [handleClick, isDisabled],
      ),
    });

    let pStyle = style;
    const res = color.colorZ.safeParse(colorVal);
    const hasCustomColor =
      res.success && (variant === "filled" || variant === "outlined");
    if (hasCustomColor) {
      const theme = Theming.use();
      pStyle = {
        ...pStyle,
        [CSS.var("btn-color")]: color.rgbString(res.data),
        [CSS.var("btn-text-color")]: color.rgbCSS(
          color.pickByContrast(res.data, theme.colors.text, theme.colors.textInverted),
        ),
      };
    }

    if (!parsedDelay.isZero)
      pStyle = {
        ...pStyle,
        [CSS.var("btn-delay")]: `${parsedDelay.seconds.toString()}s`,
      };

    if (size == null && level != null) size = Text.LEVEL_COMPONENT_SIZES[level];
    else if (size != null && level == null) level = Text.COMPONENT_SIZE_LEVELS[size];
    else size ??= "medium";

    return (
      <Text.WithIcon<"button", any>
        el="button"
        className={CSS(
          CSS.B("btn"),
          CSS.M("clickable"),
          CSS.size(size),
          CSS.sharp(sharp),
          CSS.shade(shade),
          variant !== "preview" && CSS.disabled(isDisabled),
          status != null && CSS.M(status),
          CSS.M(variant),
          hasCustomColor && CSS.BM("btn", "custom-color"),
          className,
        )}
        tabIndex={tabIndex}
        type={type}
        level={level ?? Text.COMPONENT_SIZE_LEVELS[size]}
        gap={gap}
        onClick={handleClick}
        onMouseDown={handleMouseDown}
        noWrap
        style={pStyle}
        startIcon={startIcon}
        color={colorVal}
        {...rest}
        shade={textShade}
      >
        {children}
      </Text.WithIcon>
    );
  },
);
