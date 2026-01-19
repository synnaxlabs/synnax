// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/button/Button.css";

import { color, record, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useRef } from "react";

import { CSS } from "@/css";
import { type Generic } from "@/generic";
import { Icon } from "@/icon";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";

export type ElementType = "button" | "a" | "div" | "label" | "textarea";

/** The variant of button */
export type Variant =
  | "filled"
  | "outlined"
  | "text"
  | "suggestion"
  | "preview"
  | "shadow";

export interface ExtensionProps
  extends Omit<Text.ExtensionProps, "variant">, Tooltip.WrapProps {
  variant?: Variant;
  trigger?: Triggers.Trigger;
  triggerIndicator?: boolean | Triggers.Trigger;
  textColor?: Text.TextProps["color"];
  textVariant?: Text.Variant;
  contrast?: Text.Shade | false;
  disabled?: boolean;
  preventClick?: boolean;
  propagateClick?: boolean;
  onClickDelay?: number | TimeSpan;
  ghost?: boolean;
}

/** The props for the {@link Button} component. */
export type ButtonProps<E extends ElementType = "button"> = Omit<
  Generic.OptionalElementProps<E>,
  "color"
> &
  ExtensionProps;

const MODULE_CLASS = "btn";

const resolveTriggerIndicator = (
  triggerIndicator: boolean | Triggers.Trigger | undefined,
  trigger: Triggers.Trigger | undefined,
): Triggers.Trigger | undefined => {
  if (triggerIndicator === true) return trigger;
  if (triggerIndicator != null && triggerIndicator !== false) return triggerIndicator;
  return undefined;
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
 */
const Base = <E extends ElementType = "button">({
  size,
  variant = "outlined",
  className,
  disabled,
  preventClick,
  level,
  trigger,
  triggerIndicator,
  onClickDelay = 0,
  onClick,
  color: colorVal,
  status,
  style,
  onMouseDown,
  textColor,
  textVariant,
  tabIndex,
  contrast,
  children,
  defaultEl = "button",
  el,
  ghost,
  propagateClick = false,
  href,
  ...rest
}: ButtonProps<E>): ReactElement => {
  const parsedDelay = TimeSpan.fromMilliseconds(onClickDelay);
  const isDisabled = disabled === true || status === "loading" || status === "disabled";
  // The shadow variant appears as text but shows outline on hover.
  // We don't convert it here, let CSS handle the behavior.
  if (variant === "preview") preventClick = true;

  if (disabled || (preventClick && tabIndex == null)) tabIndex = -1;

  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    if (!propagateClick) e.stopPropagation();
    if (isDisabled || variant === "preview" || preventClick === true) return;
    // @ts-expect-error - TODO: fix this
    if (parsedDelay.isZero) return onClick?.(e);
  };

  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleMouseDown = (e: any) => {
    if (tabIndex == -1) e.preventDefault();
    onMouseDown?.(e);
    if (isDisabled || variant === "preview" || parsedDelay.isZero) return;
    document.addEventListener(
      "mouseup",
      () => timeoutRef.current != null && clearTimeout(timeoutRef.current),
    );
    timeoutRef.current = setTimeout(() => {
      onClick?.(e);
      timeoutRef.current = null;
    }, parsedDelay.milliseconds);
  };

  Triggers.use({
    triggers: trigger,
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
  else if (defaultEl !== "div") size ??= "medium";
  level ??= "p";

  const isLoading = status === "loading";
  const square = Text.isSquare(children);

  const parsedTriggerIndicator = resolveTriggerIndicator(triggerIndicator, trigger);

  return (
    <Text.Text<E>
      el={el}
      defaultEl={defaultEl}
      direction="x"
      className={CSS(
        CSS.B(MODULE_CLASS),
        contrast != null && CSS.BM(MODULE_CLASS, `contrast-${contrast}`),
        preventClick === true && CSS.BM(MODULE_CLASS, "prevent-click"),
        variant !== "preview" && CSS.disabled(isDisabled),
        CSS.BM(MODULE_CLASS, variant),
        hasCustomColor && CSS.BM(MODULE_CLASS, "custom-color"),
        ghost && CSS.BM(MODULE_CLASS, "ghost"),
        className,
      )}
      size={size}
      tabIndex={tabIndex}
      onClick={handleClick}
      onMouseDown={handleMouseDown}
      style={pStyle}
      color={textColor}
      gap={size === "small" || size === "tiny" ? "small" : undefined}
      bordered={variant !== "text"}
      level={level}
      variant={textVariant}
      square={square}
      overflow="nowrap"
      status={status}
      href={href}
      {...(record.purgeUndefined(rest) as Text.TextProps<E>)}
    >
      {(!isLoading || !square) && children}
      {isLoading && <Icon.Loading />}
      {parsedTriggerIndicator != null && (
        <Triggers.Text
          className={CSS.B("trigger-indicator")}
          aria-label="trigger-indicator"
          trigger={parsedTriggerIndicator}
          color={9}
          gap="tiny"
          level={Text.downLevel(level)}
        />
      )}
    </Text.Text>
  );
};

export const Button = Tooltip.wrap(Base) as typeof Base;
