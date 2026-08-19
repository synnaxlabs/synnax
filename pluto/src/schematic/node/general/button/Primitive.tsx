// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/button/button.css";

import {
  type CSSProperties,
  type MouseEventHandler,
  type ReactElement,
  useMemo,
} from "react";

import { Button as Base } from "@/button";
import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/button/config";
import { symbolColorVar } from "@/schematic/symbolColor";

interface ButtonProps extends Omit<Config, "variant"> {
  className?: string;
  onClick?: MouseEventHandler<HTMLButtonElement>;
  onMouseDown?: MouseEventHandler<HTMLButtonElement>;
  onMouseUp?: MouseEventHandler<HTMLButtonElement>;
}

export const Button = ({
  onClick,
  onMouseDown,
  onMouseUp,
  orientation = "left",
  label,
  color,
  size,
  level,
  mode = "fire",
  onClickDelay: delay,
}: ButtonProps): ReactElement => {
  const style = useMemo<CSSProperties>(
    () => ({ [CSS.var("symbol-color")]: symbolColorVar(color) }),
    [color],
  );
  // The activation delay gates Base.Button's onClick, so single-shot actuation
  // (fire's release write, pulse's press write) routes through it. An undelayed
  // pulse keeps its raw press edge. Momentary always acts on raw press/release:
  // the hold is the actuation, so a hold delay has no meaning there.
  const delayed = (delay ?? 0) > 0;
  let handlers: Pick<ButtonProps, "onClick" | "onMouseDown" | "onMouseUp">;
  if (mode === "momentary") handlers = { onMouseDown, onMouseUp };
  else if (mode === "pulse")
    handlers = delayed ? { onClick: onMouseDown } : { onMouseDown };
  else handlers = { onClick };
  return (
    <Primitive.Div orientation={orientation}>
      <Base.Button
        variant="filled"
        className={CSS.cx(CSS.B("symbol-colored"), CSS.B("symbol-button"))}
        style={style}
        {...handlers}
        size={size}
        level={level}
        direction={label?.direction}
        onClickDelay={mode === "momentary" ? 0 : delay}
      >
        {label?.label ?? ""}
      </Base.Button>
      <Handle.Rectangle
        orientation={orientation}
        left={0}
        top={0}
        right={100}
        bottom={100}
      />
    </Primitive.Div>
  );
};
