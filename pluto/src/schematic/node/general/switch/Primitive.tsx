// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/switch/switch.css";

import { CSS } from "@synnaxlabs/lyra/css";
import { useHold } from "@synnaxlabs/lyra/hooks";
import { Input as BaseInput } from "@synnaxlabs/lyra/input";
import { blockActivation } from "@synnaxlabs/lyra/util";
import { color, location } from "@synnaxlabs/x";
import { type CSSProperties, type MouseEventHandler, type ReactElement } from "react";

import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Toggle } from "@/schematic/node/common/toggle";

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
  const colorVar = color.rgbaString(colorVal);
  const hold = useHold<HTMLElement>({ onClick, onClickDelay, disabled });
  const delayed = !hold.delay.isZero;
  const style: CSSProperties = {
    [CSS.variable("switch-scale")]: scale,
    [CSS.variable("symbol-color")]: colorVar,
    ...(delayed && {
      [CSS.variable("toggle-delay")]: `${hold.delay.seconds.toString()}s`,
    }),
  };
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
        // The diagram's key guard lets inputs through, so the checkbox blocks its own.
        onKeyDown={blockActivation}
        onKeyUp={blockActivation}
      />
      <Handle.Linear orientation={orientation} left={0} right={100} />
    </Primitive.Div>
  );
};
