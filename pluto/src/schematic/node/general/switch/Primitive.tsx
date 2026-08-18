// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/switch/switch.css";

import { type CSSProperties, type MouseEventHandler, type ReactElement } from "react";

import { CSS } from "@/css";
import { Input as BaseInput } from "@/input";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Toggle } from "@/schematic/node/common/toggle";
import { symbolColorVar } from "@/schematic/symbolColor";

export interface Props extends Omit<Toggle.ButtonProps, "onClick"> {
  onClick?: MouseEventHandler<HTMLElement>;
}

export const Switch = ({
  enabled = false,
  onClick,
  orientation = "left",
  color: colorVal,
}: Props): ReactElement => {
  const colorVar = symbolColorVar(colorVal);
  const style: CSSProperties | undefined =
    colorVar != null ? { [CSS.var("symbol-color")]: colorVar } : undefined;
  return (
    <Primitive.Div
      orientation={orientation}
      className={CSS.cx(
        colorVar != null && CSS.B("symbol-colored"),
        colorVar != null && CSS.BM("switch-symbol", "colored"),
      )}
      style={style}
    >
      <BaseInput.Switch value={enabled} onClick={onClick} onChange={() => {}} />
      <Handle.Linear orientation={orientation} left={0} right={100} />
    </Primitive.Div>
  );
};
