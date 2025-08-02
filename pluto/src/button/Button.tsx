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
import { TimeSpan } from "@synnaxlabs/x/telem";
import {
  Children,
  type ReactElement,
  type ReactNode,
  useCallback,
  useRef,
} from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { type Generic } from "@/generic";
import { Icon } from "@/icon";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";

export type ElementType = "button" | "a" | "div";

/** The variant of button */
export type Variant =
  | "filled"
  | "outlined"
  | "text"
  | "suggestion"
  | "preview"
  | "shadow";

export interface ExtensionProps extends Text.ExtensionProps, Tooltip.WrapProps {
  variant?: Variant;
  size?: Component.Size;
  sharp?: boolean;
  loading?: boolean;
  trigger?: Triggers.Trigger;
  status?: status.Variant;
  textColor?: Text.Shade;
  contrast?: Text.Shade;
  disabled?: boolean;
  allowClick?: boolean;
  onClickDelay?: number | TimeSpan;
}

/** The props for the {@link Button} component. */
export type ButtonProps<E extends ElementType = "button"> = Omit<
  Generic.OptionalElementProps<E>,
  "color"
> &
  ExtensionProps;

const isIconOnly = (children: ReactNode): boolean => {
  if (Children.count(children) !== 1) return false;
  if (typeof children === "string") return children.length === 1;
  return true;
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
const Core = <E extends ElementType = "button">({
  size,
  variant = "outlined",
  className,
  sharp = false,
  disabled = false,
  allowClick = true,
  loading = false,
  level,
  trigger: triggers,
  onClickDelay = 0,
  onClick,
  color: colorVal,
  status,
  style,
  onMouseDown,
  textColor,
  tabIndex,
  contrast,
  children,
  ...rest
}: ButtonProps<E>): ReactElement => {
  const parsedDelay = TimeSpan.fromMilliseconds(onClickDelay);
  const isDisabled = disabled || loading;
  // We implement the shadow variant to maintain compatibility with the input
  // component API.
  if (variant == "shadow") variant = "text";

  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    if (isDisabled || variant === "preview") return;
    // @ts-expect-error - TODO: fix this
    if (parsedDelay.isZero) return onClick?.(e);
  };

  const toRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleMouseDown = (e: any) => {
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

  const iconOnly = isIconOnly(children);

  return (
    <Text.Text<E>
      direction="x"
      className={CSS(
        CSS.B("btn"),
        iconOnly && CSS.BM("btn", "icon"),
        allowClick && CSS.BM("btn", `shade-${contrast}`),
        CSS.sharp(sharp),
        CSS.height(size),
        variant !== "preview" && CSS.disabled(isDisabled),
        status != null && CSS.M(status),
        CSS.M(variant),
        hasCustomColor && CSS.BM("btn", "custom-color"),
        className,
      )}
      tabIndex={tabIndex}
      onClick={handleClick}
      onMouseDown={handleMouseDown}
      noWrap
      style={pStyle}
      color={colorVal}
      gap={size === "small" || size === "tiny" ? "small" : "medium"}
      {...(rest as Text.TextProps<E>)}
      el={rest.el ?? "button"}
      level={level}
    >
      {children}
      {loading && <Icon.Loading />}
    </Text.Text>
  );
};

export const Button = Tooltip.wrap(Core) as typeof Core;
