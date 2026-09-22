// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/button/Button.css";

import { color, record, text, type TimeSpan } from "@synnaxlabs/x";
import {
  type KeyboardEvent as ReactKeyboardEvent,
  type KeyboardEventHandler,
  type MouseEvent as ReactMouseEvent,
  type ReactElement,
  useCallback,
  useMemo,
  useRef,
} from "react";

import { SIZE_TEXT_LEVELS, TEXT_LEVEL_SIZES } from "@/component/text";
import { CSS } from "@/css";
import { type Generic } from "@/generic";
import { useCombinedRefs, useHold, type UseHoldProps } from "@/hooks";
import { Icon } from "@/icon";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";

/** The elements a Button can render as. `a` makes it a link, `label` a form control. */
export type ElementType = "button" | "a" | "div" | "label" | "textarea";

/** The rest-state emphasis of the button chassis. */
export type Variant = "filled" | "outlined" | "text";

/** The button-specific props {@link ButtonProps} adds to its element's own props. */
export interface ExtensionProps
  extends Omit<Text.ExtensionProps, "variant">, Tooltip.WrapProps {
  /** The rest-state emphasis. Defaults to "outlined". */
  variant?: Variant;
  /** A keyboard trigger that clicks the button while it is mounted. */
  trigger?: Triggers.Trigger;
  /** Renders the trigger's keys beside the label. true shows `trigger`; a trigger of
   * its own shows that instead, for a button whose real shortcut lives elsewhere. */
  triggerIndicator?: boolean | Triggers.Trigger;
  /** Overrides the label color without moving the chassis off its variant. */
  textColor?: Text.TextProps["color"];
  /** The text variant of the label. */
  textVariant?: Text.Variant;
  /** Blocks interaction and dims the button. */
  disabled?: boolean;
  /** Renders the button flat and inert, for a preview of an interface. */
  preview?: boolean;
  /** Swallows the click without calling onClick. Use for a button that is momentarily
   * inapplicable but must not read as disabled. */
  preventClick?: boolean;
  /** Lets the click reach an ancestor's handler. Clicks stop at the button otherwise. */
  propagateClick?: boolean;
  /** Holds onClick until the button has been held this long, filling a progress bar
   * meanwhile. Use it to guard a destructive action. */
  onClickDelay?: number | TimeSpan;
  /** Marks the button as a hidden action its pluto--reveals container shows. */
  reveal?: boolean;
}

/** The props for the {@link Button} component. */
export type ButtonProps<E extends ElementType = "button"> = Omit<
  Generic.OptionalElementProps<E>,
  "color" | "onClick" | "onMouseDown" | "onKeyDown" | "onKeyUp"
> &
  ExtensionProps &
  Pick<UseHoldProps<HTMLElement>, "onClick" | "onMouseDown"> & {
    onKeyDown?: KeyboardEventHandler<HTMLElement>;
    onKeyUp?: KeyboardEventHandler<HTMLElement>;
  };

const MODULE_CLASS = "btn";

const resolveTriggerIndicator = (
  triggerIndicator: boolean | Triggers.Trigger | undefined,
  trigger: Triggers.Trigger | undefined,
): Triggers.Trigger | undefined => {
  if (triggerIndicator === true) return trigger;
  if (triggerIndicator != null && triggerIndicator !== false) return triggerIndicator;
  return undefined;
};

const FOCUSABLE =
  'a[href], button, input, select, textarea, [contenteditable="true"], [tabindex]';

/**
 * The standard clickable. Renders as a `button` unless `el` names another
 * {@link ElementType}, carries an optional keyboard trigger and tooltip, and lays its
 * icons and label out on the shared size scale.
 *
 * @example <Button.Button onClick={save}><Icon.Save />Save</Button.Button>
 * @example <Button.Button variant="text" trigger={["Control", "S"]} triggerIndicator />
 */
export const Button = <E extends ElementType = "button">(
  props: ButtonProps<E>,
): ReactElement => {
  const {
    size: sizeProp,
    variant = "outlined",
    className,
    disabled,
    preview,
    preventClick: preventClickProp,
    level: levelProp,
    trigger,
    triggerIndicator,
    onClickDelay = 0,
    onClick,
    onKeyDown,
    onKeyUp,
    color: colorVal,
    status,
    style,
    onMouseDown,
    textColor,
    textVariant,
    tabIndex: tabIndexProp,
    children,
    defaultEl = "button",
    el,
    reveal,
    propagateClick = false,
    draggable,
    href,
    ref,
    tooltip,
    tooltipLocation,
    hideTooltip,
    ...rest
  }: ButtonProps<ElementType> = props;
  const elRef = useRef<HTMLElement>(null);
  const combinedRef = useCombinedRefs<HTMLElement>(ref, elRef);
  const isDisabled = disabled === true || status === "loading" || status === "disabled";
  const preventClick = preventClickProp === true || preview === true;
  const hold = useHold({
    onClick,
    onMouseDown,
    onClickDelay,
    disabled: isDisabled || preview === true,
  });

  const tabIndex =
    disabled || (preventClick && tabIndexProp == null) ? -1 : tabIndexProp;

  const handleClick = (e: ReactMouseEvent<HTMLElement>) => {
    if (!propagateClick) e.stopPropagation();
    if (isDisabled || preview === true || preventClick) return;
    hold.onClick(e);
  };

  // A non-button chassis has no native Enter/Space activation, so a focusable one gets
  // it from the component. tabIndex -1 still counts: roving-tabindex tabs hold focus
  // programmatically. The target guard keeps keystrokes on nested interactives (inputs,
  // editables) from activating the chassis.
  const resolvedEl = Text.parseElement(levelProp, el, defaultEl, textVariant, href);
  const ownsActivation =
    (resolvedEl === "div" || resolvedEl === "label") && tabIndex != null;
  const handleKeyDown = (e: ReactKeyboardEvent<HTMLElement>) => {
    onKeyDown?.(e);
    hold.onKeyDown(e);
    if (!ownsActivation || e.defaultPrevented) return;
    if (e.target !== e.currentTarget) return;
    if (e.key !== "Enter" && e.key !== " ") return;
    e.preventDefault();
    if (!propagateClick) e.stopPropagation();
    e.currentTarget.click();
  };

  const handleKeyUp = (e: ReactKeyboardEvent<HTMLElement>) => {
    onKeyUp?.(e);
    hold.onKeyUp(e);
  };

  const handleMouseDown = (e: ReactMouseEvent<HTMLElement>) => {
    // Preventing default on mousedown cancels a native dragstart, so skip it for
    // draggable buttons (e.g. roving-tabindex tabs that are also drag sources). The
    // cancelled default also moves focus, so skip it when a focusable descendant owns
    // the press: the chassis is not the element the browser would focus.
    if (
      tabIndex == -1 &&
      draggable !== true &&
      e.target instanceof Element &&
      e.target.closest(FOCUSABLE) === e.currentTarget
    )
      e.preventDefault();
    hold.onMouseDown(e);
  };

  Triggers.use({
    triggers: trigger,
    callback: useCallback<(e: Triggers.UseEvent) => void>(
      ({ stage }) => {
        if (stage !== "end" || isDisabled || preview === true) return;
        elRef.current?.click();
      },
      [isDisabled, preview],
    ),
  });

  const res = color.colorZ.safeParse(colorVal);
  const hasCustomColor =
    res.success && (variant === "filled" || variant === "outlined");
  const theme = Theming.use();

  const pStyle = useMemo(() => {
    let s = style;
    if (hasCustomColor)
      s = {
        ...s,
        [CSS.variable("btn-color")]: color.rgbString(res.data),
        [CSS.variable("btn-text-color")]: color.rgbCSS(
          color.pickByContrast(res.data, theme.colors.text, theme.colors.textInverted),
        ),
      };
    if (!hold.delay.isZero)
      s = {
        ...s,
        [CSS.variable("btn-delay")]: `${hold.delay.seconds.toString()}s`,
      };
    return s;
  }, [style, hasCustomColor, colorVal, theme, hold.delay]);

  let size = sizeProp;
  let level = levelProp;
  if (size == null && level != null) size = TEXT_LEVEL_SIZES[level];
  else if (size != null && level == null) level = SIZE_TEXT_LEVELS[size];
  else if (defaultEl !== "div") size ??= "medium";
  level ??= "p";

  const isLoading = status === "loading";
  const square = Text.isSquare(children);

  const parsedTriggerIndicator = resolveTriggerIndicator(triggerIndicator, trigger);

  const element = (
    <Text.Text
      el={el}
      defaultEl={defaultEl}
      direction="x"
      className={CSS.cls(
        CSS.B(MODULE_CLASS),
        preventClick && CSS.BM(MODULE_CLASS, "prevent-click"),
        !preview && CSS.disabled(isDisabled),
        CSS.BM(MODULE_CLASS, variant),
        preview === true && CSS.BM(MODULE_CLASS, "preview"),
        hasCustomColor && CSS.BM(MODULE_CLASS, "custom-color"),
        reveal === true && CSS.M("reveal"),
        hold.pressed && CSS.M("pressed"),
        className,
      )}
      size={size}
      tabIndex={tabIndex}
      aria-disabled={isDisabled || undefined}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      onKeyUp={handleKeyUp}
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
      draggable={draggable}
      ref={combinedRef}
      {...record.purgeUndefined(rest)}
    >
      {(!isLoading || !square) && children}
      {isLoading && <Icon.Loading />}
      {parsedTriggerIndicator != null && (
        <Triggers.Text
          className={CSS.B("trigger-indicator")}
          aria-label="trigger-indicator"
          aria-hidden
          trigger={parsedTriggerIndicator}
          color={9}
          gap="tiny"
          level={text.downLevel(level)}
        />
      )}
    </Text.Text>
  );
  if (tooltip == null) return element;
  return (
    <Tooltip.Dialog location={tooltipLocation} hide={hideTooltip}>
      {tooltip}
      {element}
    </Tooltip.Dialog>
  );
};
