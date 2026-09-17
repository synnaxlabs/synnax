// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type CrudeTimeSpan } from "@synnaxlabs/x";
import {
  type ComponentPropsWithRef,
  type KeyboardEventHandler,
  type ReactElement,
  useMemo,
} from "react";

import { CSS } from "@/css";
import { Keyboard } from "@/schematic/node/common/keyboard";
import { Primitive } from "@/schematic/node/common/primitive";
import { type OrientableProps } from "@/schematic/node/common/primitive/orientable";
import { useHold } from "@/schematic/node/common/toggle/hold";

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
  children,
  ...rest
}: ButtonProps): ReactElement => {
  const hold = useHold<HTMLButtonElement>({ onClick, onMouseDown, onClickDelay });
  const delayed = !hold.delay.isZero;

  const handleKeyDown: KeyboardEventHandler<HTMLButtonElement> = (e) => {
    onKeyDown?.(e);
    Keyboard.blockActivation(e);
  };

  const handleKeyUp: KeyboardEventHandler<HTMLButtonElement> = (e) => {
    onKeyUp?.(e);
    Keyboard.blockActivation(e);
  };

  const pStyle = useMemo(() => {
    if (!delayed) return style;
    return {
      ...style,
      [CSS.variable("toggle-delay")]: `${hold.delay.seconds.toString()}s`,
    };
  }, [hold.delay.milliseconds, style]);

  return (
    <button
      className={CSS.cls(
        CSS.B("symbol-primitive"),
        CSS.B("symbol-primitive-toggle"),
        delayed && CSS.BM("symbol-primitive-toggle", "delayed"),
        orientation != null && CSS.loc(orientation),
        enabled && CSS.M("enabled"),
        triggered && CSS.M("triggered"),
        hold.pressed && CSS.M("pressed"),
        className,
      )}
      color={color.cssString(colorVal)}
      onClick={hold.onClick}
      onMouseDown={hold.onMouseDown}
      onKeyDown={handleKeyDown}
      onKeyUp={handleKeyUp}
      style={pStyle}
      {...rest}
    >
      <Primitive.HoldFill value={delayed}>{children}</Primitive.HoldFill>
    </button>
  );
};
