// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x";
import {
  type ComponentPropsWithRef,
  type KeyboardEventHandler,
  type MouseEventHandler,
  type ReactElement,
  useMemo,
  useRef,
} from "react";

import { CSS } from "@/css";
import { type OrientableProps } from "@/schematic/node/common/primitive/orientable";

export interface ButtonBaseProps extends Omit<
  ComponentPropsWithRef<"button">,
  "color" | "value"
> {
  triggered?: boolean;
  enabled?: boolean;
  color?: color.Crude;
  onClickDelay?: CrudeTimeSpan;
}

export interface ButtonProps extends ButtonBaseProps, OrientableProps {}

const isActivationKey = (key: string): boolean => key === " " || key === "Enter";

export const Button = ({
  className,
  enabled = false,
  triggered = false,
  orientation = "left",
  color: colorVal,
  onClickDelay = 0,
  onClick,
  onMouseDown,
  onKeyDown,
  onKeyUp,
  style,
  ...rest
}: ButtonProps): ReactElement => {
  const parsedDelay = TimeSpan.fromMilliseconds(onClickDelay);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleClick: MouseEventHandler<HTMLButtonElement> = (e) => {
    if (parsedDelay.isZero) onClick?.(e);
  };

  const handleMouseDown: MouseEventHandler<HTMLButtonElement> = (e) => {
    onMouseDown?.(e);
    if (parsedDelay.isZero) return;
    document.addEventListener(
      "mouseup",
      () => {
        if (timeoutRef.current != null) clearTimeout(timeoutRef.current);
        timeoutRef.current = null;
      },
      { once: true },
    );
    timeoutRef.current = setTimeout(() => {
      onClick?.(e);
      timeoutRef.current = null;
    }, parsedDelay.milliseconds);
  };

  // A clicked button keeps focus, so a later Space or Enter would actuate it through
  // the browser's synthetic click. Enter clicks on keydown and Space on keyup.
  const handleKeyDown: KeyboardEventHandler<HTMLButtonElement> = (e) => {
    onKeyDown?.(e);
    if (isActivationKey(e.key)) e.preventDefault();
  };

  const handleKeyUp: KeyboardEventHandler<HTMLButtonElement> = (e) => {
    onKeyUp?.(e);
    if (isActivationKey(e.key)) e.preventDefault();
  };

  const pStyle = useMemo(() => {
    if (parsedDelay.isZero) return style;
    return {
      ...style,
      [CSS.variable("toggle-delay")]: `${parsedDelay.seconds.toString()}s`,
    };
  }, [parsedDelay.milliseconds, style]);

  return (
    <button
      className={CSS.cls(
        CSS.B("symbol-primitive"),
        CSS.B("symbol-primitive-toggle"),
        !parsedDelay.isZero && CSS.BM("symbol-primitive-toggle", "delayed"),
        orientation != null && CSS.loc(orientation),
        enabled && CSS.M("enabled"),
        triggered && CSS.M("triggered"),
        className,
      )}
      color={color.cssString(colorVal)}
      onClick={handleClick}
      onMouseDown={handleMouseDown}
      onKeyDown={handleKeyDown}
      onKeyUp={handleKeyUp}
      style={pStyle}
      {...rest}
    />
  );
};
