// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/switch/switch.css";

import { location } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type MouseEventHandler,
  type ReactElement,
  useMemo,
} from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Input as BaseInput } from "@/input";
import { Handle } from "@/schematic/node/common/handle";
import { Keyboard } from "@/schematic/node/common/keyboard";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Toggle } from "@/schematic/node/common/toggle";
import { symbolColorVar } from "@/schematic/symbolColor";

export interface Props extends Omit<Toggle.ButtonProps, "onClick" | "onMouseDown"> {
  onClick?: MouseEventHandler<HTMLElement>;
  scale?: number;
}

export const Switch = ({
  enabled = false,
  onClick,
  onClickDelay,
  orientation = "left",
  color: colorVal,
  scale = 1,
  disabled,
}: Props): ReactElement => {
  const colorVar = symbolColorVar(colorVal);
  const hold = Button.useHold<HTMLElement>({ onClick, onClickDelay, disabled });
  const delayed = !hold.delay.isZero;
  const style = useMemo<CSSProperties>(
    () => ({
      [CSS.variable("switch-scale")]: scale,
      [CSS.variable("symbol-color")]: colorVar,
      ...(delayed && {
        [CSS.variable("toggle-delay")]: `${hold.delay.seconds.toString()}s`,
      }),
    }),
    [scale, colorVar, delayed, hold.delay.milliseconds],
  );
  return (
    <Primitive.Div
      orientation={orientation}
      className={CSS.cls(
        colorVar != null && CSS.B("symbol-colored"),
        colorVar != null && CSS.BM("switch-symbol", "colored"),
        delayed && CSS.BM("switch-symbol", "delayed"),
        hold.pressed && CSS.M("pressed"),
        CSS.dir(location.direction(orientation)),
      )}
      style={style}
      onMouseDown={hold.onMouseDown}
    >
      <BaseInput.Switch
        value={enabled}
        disabled={disabled}
        onClick={hold.onClick}
        onChange={() => {}}
        onKeyDown={Keyboard.blockActivation}
        onKeyUp={Keyboard.blockActivation}
      />
      <Handle.Linear orientation={orientation} left={0} right={100} />
    </Primitive.Div>
  );
};
