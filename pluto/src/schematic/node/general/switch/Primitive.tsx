// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/switch/switch.css";

import { useHold } from "@synnaxlabs/lyra/hooks";
import { location } from "@synnaxlabs/x";
import { type CSSProperties, type MouseEventHandler, type ReactElement } from "react";

import { CSS } from "@/css";
import { Custom } from "@/schematic/node/common/custom";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Toggle } from "@/schematic/node/common/toggle";
import svg from "@/schematic/node/general/switch/Switch.svg?raw";

const SPEC = Custom.createBuiltIn({ svg, handles: [] });

const SWITCH_SCALE = CSS.variable("switch-scale");

export interface Props
  extends Omit<Toggle.ButtonProps, "onClick" | "onMouseDown">, Primitive.SVGBasedProps {
  onClick?: MouseEventHandler<HTMLElement>;
}

export const Switch = ({
  enabled = false,
  onClick,
  onClickDelay,
  orientation = "left",
  strokeColor,
  scale = 1,
  disabled,
}: Props): ReactElement => {
  const hold = useHold<HTMLElement>({ onClick, onClickDelay, disabled });
  const delayed = !hold.delay.isZero;
  const style: CSSProperties = {
    [SWITCH_SCALE]: Primitive.resolveScale(scale),
    ...(delayed && {
      [Primitive.TOGGLE_DELAY]: `${hold.delay.seconds.toString()}s`,
    }),
  };
  return (
    <Primitive.Div
      orientation={orientation}
      className={CSS.cls(
        CSS.B("switch-symbol"),
        enabled && CSS.BM("switch-symbol", "on"),
        strokeColor != null && CSS.BM("switch-symbol", "colored"),
        disabled === true && CSS.BM("switch-symbol", "disabled"),
        delayed && CSS.BM("switch-symbol", "delayed"),
        hold.pressed && CSS.M("pressed"),
      )}
      style={style}
      onMouseDown={hold.onMouseDown}
      onClick={disabled === true ? undefined : hold.onClick}
    >
      <Custom.Drawing
        spec={SPEC}
        builtIn
        strokeColor={strokeColor}
        // The switch turns only to stand upright, with off at the bottom.
        orientation={location.direction(orientation) === "y" ? "top" : "left"}
        scale={scale}
      />
      <Handle.Linear orientation={orientation} left={0} right={100} />
    </Primitive.Div>
  );
};
