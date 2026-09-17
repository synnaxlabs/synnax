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
  useState,
} from "react";

import { CSS } from "@/css";
import { Keyboard } from "@/schematic/node/common/keyboard";
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
  // WebKit sets :active on a secondary press, so pressed styling follows this flag.
  const [pressed, setPressed] = useState(false);

  const handleClick: MouseEventHandler<HTMLButtonElement> = (e) => {
    if (parsedDelay.isZero) onClick?.(e);
  };

  const handleMouseDown: MouseEventHandler<HTMLButtonElement> = (e) => {
    onMouseDown?.(e);
    if (e.button !== 0) return;
    setPressed(true);
    document.addEventListener(
      "mouseup",
      () => {
        setPressed(false);
        if (timeoutRef.current != null) clearTimeout(timeoutRef.current);
        timeoutRef.current = null;
      },
      { once: true },
    );
    if (parsedDelay.isZero) return;
    timeoutRef.current = setTimeout(() => {
      onClick?.(e);
      timeoutRef.current = null;
    }, parsedDelay.milliseconds);
  };

  const handleKeyDown: KeyboardEventHandler<HTMLButtonElement> = (e) => {
    onKeyDown?.(e);
    Keyboard.blockActivation(e);
  };

  const handleKeyUp: KeyboardEventHandler<HTMLButtonElement> = (e) => {
    onKeyUp?.(e);
    Keyboard.blockActivation(e);
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
        pressed && CSS.M("pressed"),
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
